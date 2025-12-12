import { CustomObjectsApi } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from '../utils/logger';
import { BaseController, BaseResource, BaseStatus } from './base-controller';

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

interface CloudflareRulesetStatus extends BaseStatus {
  rulesetId?: string;
}

type CloudflareRulesetResource = BaseResource<CloudflareRulesetSpec, CloudflareRulesetStatus>;

const GROUP = 'cloudflare.k8s.io';
const VERSION = 'v1';
const PLURAL = 'cloudflare-rulesets';
const KIND = 'CloudflareRuleset';

export class CloudflareRulesetController extends BaseController<CloudflareRulesetSpec, CloudflareRulesetStatus> {
  constructor(k8sApi: CustomObjectsApi, cloudflare: Cloudflare) {
    super(k8sApi, cloudflare, GROUP, VERSION, PLURAL, KIND);
  }

  protected async reconcile(resource: CloudflareRulesetResource): Promise<void> {
    const { metadata, spec, status } = resource;
    logger.info(`Reconciling ${KIND} ${metadata.namespace}/${metadata.name}`);

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
          lastSync: new Date().toISOString(),
          observedGeneration: metadata.generation
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
          lastSync: new Date().toISOString(),
          observedGeneration: metadata.generation
        });
      }
    } catch (error) {
      logger.error('Failed to reconcile ruleset:', error);
      throw error;
    }
  }

  protected async delete(resource: CloudflareRulesetResource): Promise<void> {
    const { metadata, spec, status } = resource;
    logger.info(`Deleting ${KIND} ${metadata.namespace}/${metadata.name}`);

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
}
