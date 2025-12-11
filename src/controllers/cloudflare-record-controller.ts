import { CustomObjectsApi, Watch } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from '../utils/logger';

interface CloudflareRecordSpec {
  zoneId: string;
  name: string;
  type: string;
  content: string;
  ttl?: number;
  proxied?: boolean;
  priority?: number;
  comment?: string;
}

interface CloudflareRecordStatus {
  recordId?: string;
  state: 'Pending' | 'Active' | 'Error';
  message?: string;
  lastSync?: string;
}

interface CloudflareRecordResource {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace: string;
    uid?: string;
    resourceVersion?: string;
  };
  spec: CloudflareRecordSpec;
  status?: CloudflareRecordStatus;
}

const GROUP = 'cloudflare.example.com';
const VERSION = 'v1';
const PLURAL = 'cloudflarerecords';

export class CloudflareRecordController {
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
        async (type, resource: CloudflareRecordResource) => {
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

  private async handleEvent(type: string, resource: CloudflareRecordResource) {
    const { metadata, spec } = resource;
    logger.info(`Event ${type} for CloudflareRecord ${metadata.namespace}/${metadata.name}`);

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

  private async reconcile(resource: CloudflareRecordResource) {
    const { metadata, spec, status } = resource;
    logger.info(`Reconciling CloudflareRecord ${metadata.namespace}/${metadata.name}`);

    try {
      // Check if record already exists
      if (status?.recordId) {
        // Update existing record
        logger.info(`Updating DNS record ${status.recordId} in zone ${spec.zoneId}`);

        await this.cloudflare.dns.records.edit(
          status.recordId,
          {
            zone_id: spec.zoneId,
            name: spec.name,
            type: spec.type as any,
            content: spec.content,
            ttl: spec.ttl || 1,
            proxied: spec.proxied || false,
            priority: spec.priority,
            comment: spec.comment
          }
        );

        await this.updateStatus(resource, {
          recordId: status.recordId,
          state: 'Active',
          message: 'DNS record updated successfully',
          lastSync: new Date().toISOString()
        });
      } else {
        // Create new record
        logger.info(`Creating DNS record ${spec.name} in zone ${spec.zoneId}`);

        const response: any = await this.cloudflare.dns.records.create({
          zone_id: spec.zoneId,
          name: spec.name,
          type: spec.type as any,
          content: spec.content,
          ttl: spec.ttl || 1,
          proxied: spec.proxied || false,
          priority: spec.priority,
          comment: spec.comment
        });

        await this.updateStatus(resource, {
          recordId: response.id,
          state: 'Active',
          message: 'DNS record created successfully',
          lastSync: new Date().toISOString()
        });
      }
    } catch (error) {
      logger.error('Failed to reconcile DNS record:', error);
      throw error;
    }
  }

  private async delete(resource: CloudflareRecordResource) {
    const { metadata, spec, status } = resource;
    logger.info(`Deleting CloudflareRecord ${metadata.namespace}/${metadata.name}`);

    if (!status?.recordId) {
      logger.warn('No recordId found, skipping deletion');
      return;
    }

    try {
      await this.cloudflare.dns.records.delete(
        status.recordId,
        { zone_id: spec.zoneId }
      );
      logger.info(`DNS record ${status.recordId} deleted successfully`);
    } catch (error) {
      logger.error('Failed to delete DNS record:', error);
      throw error;
    }
  }

  private async updateStatus(
    resource: CloudflareRecordResource,
    status: CloudflareRecordStatus
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
