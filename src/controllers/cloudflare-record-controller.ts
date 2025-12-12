import { CustomObjectsApi } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from '../utils/logger';
import { BaseController, BaseResource, BaseStatus } from './base-controller';

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

interface CloudflareRecordStatus extends BaseStatus {
  recordId?: string;
}

type CloudflareRecordResource = BaseResource<CloudflareRecordSpec, CloudflareRecordStatus>;

const GROUP = 'cloudflare.k8s.io';
const VERSION = 'v1';
const PLURAL = 'cloudflare-records';
const KIND = 'CloudflareRecord';

export class CloudflareRecordController extends BaseController<CloudflareRecordSpec, CloudflareRecordStatus> {
  constructor(k8sApi: CustomObjectsApi, cloudflare: Cloudflare) {
    super(k8sApi, cloudflare, GROUP, VERSION, PLURAL, KIND);
  }

  protected async reconcile(resource: CloudflareRecordResource): Promise<void> {
    const { metadata, spec, status } = resource;
    logger.info(`Reconciling ${KIND} ${metadata.namespace}/${metadata.name}`);

    try {
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
          message: 'DNS record synchronized successfully',
          lastSync: new Date().toISOString(),
          observedGeneration: metadata.generation
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
          lastSync: new Date().toISOString(),
          observedGeneration: metadata.generation
        });
      }
    } catch (error) {
      logger.error('Failed to reconcile DNS record:', error);
      throw error;
    }
  }

  protected async delete(resource: CloudflareRecordResource): Promise<void> {
    const { metadata, spec, status } = resource;
    logger.info(`Deleting ${KIND} ${metadata.namespace}/${metadata.name}`);

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
}
