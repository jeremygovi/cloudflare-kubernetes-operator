import { CustomObjectsApi, Watch } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from '../utils/logger';

interface RuleActionParameters {
  [key: string]: any;
}

interface Rule {
  action: string;
  expression: string;
  description?: string;
  enabled?: boolean;
  actionParameters?: RuleActionParameters;
}

interface CloudflareRulesetSpec {
  zoneId: string;
  name?: string;
  description?: string;
  phase: string;
  rules: Rule[];
}

interface CloudflareRulesetStatus {
  rulesetId?: string;
  state: 'Pending' | 'Active' | 'Error';
  message?: string;
  lastSync?: string;
}

interface CloudflareRulesetResource {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    uid?: string;
    resourceVersion?: string;
  };
  spec: CloudflareRulesetSpec;
  status?: CloudflareRulesetStatus;
}

const GROUP = 'cloudflare.example.com';
const VERSION = 'v1';
const PLURAL = 'cloudflare-rulesets';

export class CloudflareRulesetController {
  private watcher?: Watch;
  private abortController?: AbortController;

  constructor(
    private k8sApi: CustomObjectsApi,
    private cloudflare: Cloudflare
  ) {}

  async watch() {
    const kc = new (require('@kubernetes/client-node').KubeConfig)();
    if (process.env.KUBERNETES_SERVICE_HOST) {
      kc.loadFromCluster();
    } else {
      kc.loadFromDefault();
    }

    this.watcher = new Watch(kc);
    this.abortController = new AbortController();

    const listPath = `/apis/${GROUP}/${VERSION}/${PLURAL}`;

    logger.info(`Starting watch on ${listPath}`);

    try {
      await this.watcher.watch(
        listPath,
        {},
        async (type, resource: CloudflareRulesetResource) => {
          try {
            await this.handleEvent(type, resource);
          } catch (error) {
            logger.error('Error handling event:', error);
          }
        },
        (err) => {
          if (err) {
            logger.error('Watch error:', err);
            // Restart watch after error
            setTimeout(() => this.watch(), 5000);
          }
        }
      );
    } catch (error) {
      logger.error('Failed to start watch:', error);
      throw error;
    }
  }

  private async handleEvent(type: string, resource: CloudflareRulesetResource) {
    const { metadata, spec } = resource;
    logger.info(`Event ${type} for CloudflareRuleset ${metadata.namespace}/${metadata.name}`);
    try {
      switch (type) {
        case 'ADDED':
        case 'MODIFIED':
          await this.reconcile(resource);
          break;
        case 'DELETED':
          await this.delete(resource);
          break;
        default:
          logger.warn(`Unknown event type: ${type}`);
      }
    } catch (error) {
      logger.error(`Failed to handle ${type} event:`, error);
      await this.updateStatus(resource, {
        state: 'Error',
        message: error instanceof Error ? error.message : 'Unknown error'
      });
    }
  }

  private async reconcile(resource: CloudflareRulesetResource) {
    const { metadata, spec, status } = resource;
    logger.info(`Reconciling CloudflareRuleset ${metadata.namespace}/${metadata.name}`);
    try {
      const rulesetData = {
        name: spec.name || `k8s-${metadata.name}`,
        description: spec.description || `Managed by Kubernetes: ${metadata.namespace}/${metadata.name}`,
        kind: 'zone',
        phase: spec.phase,
        rules: spec.rules.map(rule => ({
          action: rule.action,
          expression: rule.expression,
          description: rule.description,
          enabled: rule.enabled !== undefined ? rule.enabled : true,
          action_parameters: rule.actionParameters
        }))
      };

      // Check if ruleset already exists
      if (status?.rulesetId) {
        // Update existing ruleset
        logger.info(`Updating ruleset ${status.rulesetId} in zone ${spec.zoneId}`);
        await (this.cloudflare as any).zones.rulesets.update(
          spec.zoneId,
          status.rulesetId,
          rulesetData
        );

        await this.updateStatus(resource, {
          rulesetId: status.rulesetId,
          state: 'Active',
          message: 'Ruleset updated successfully',
          lastSync: new Date().toISOString()
        });
      } else {
        // Create new ruleset
        logger.info(`Creating ruleset in zone ${spec.zoneId}`);
        const response: any = await (this.cloudflare as any).zones.rulesets.create(
          spec.zoneId,
          rulesetData
        );

        await this.updateStatus(resource, {
          rulesetId: response.id,
          state: 'Active',
          message: 'Ruleset created successfully',
          lastSync: new Date().toISOString()
        });
      }
    } catch (error) {
      logger.error('Failed to reconcile ruleset:', error);
      throw error;
    }
  }

  private async delete(resource: CloudflareRulesetResource) {
    const { metadata, spec, status } = resource;
    logger.info(`Deleting CloudflareRuleset ${metadata.namespace}/${metadata.name}`);

    if (!status?.rulesetId) {
      logger.warn('No rulesetId found, skipping deletion');
      return;
    }

    try {
      await (this.cloudflare as any).zones.rulesets.delete(
        spec.zoneId,
        status.rulesetId
      );
      logger.info(`Ruleset ${status.rulesetId} deleted successfully`);
    } catch (error) {
      logger.error('Failed to delete ruleset:', error);
      throw error;
    }
  }

  private async updateStatus(
    resource: CloudflareRulesetResource,
    status: CloudflareRulesetStatus
  ) {
    const { metadata } = resource;

    try {
      await this.k8sApi.patchNamespacedCustomObjectStatus(
        GROUP,
        VERSION,
        metadata.namespace,
        PLURAL,
        metadata.name,
        { status },
        undefined,
        undefined,
        undefined,
        { headers: { 'Content-Type': 'application/merge-patch+json' } }
      );
      logger.debug(`Status updated for ${metadata.namespace}/${metadata.name}`);
    } catch (error) {
      logger.error('Failed to update status:', error);
    }
  }

  stop() {
    if (this.abortController) {
      this.abortController.abort();
    }
  }
}
