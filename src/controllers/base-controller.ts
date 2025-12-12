import { CustomObjectsApi, Watch } from '@kubernetes/client-node';
import Cloudflare from 'cloudflare';
import { logger } from '../utils/logger';

export interface ResourceMetadata {
  name: string;
  namespace: string;
  uid?: string;
  resourceVersion?: string;
  generation?: number;
}

export interface BaseResource<TSpec, TStatus> {
  apiVersion: string;
  kind: string;
  metadata: ResourceMetadata;
  spec: TSpec;
  status?: TStatus;
}

export interface BaseStatus {
  state: 'Pending' | 'Active' | 'Error';
  message?: string;
  lastSync?: string;
  observedGeneration?: number;
}

export abstract class BaseController<TSpec, TStatus extends BaseStatus> {
  protected watcher?: Watch;
  protected abortController?: AbortController;
  private lastResourceVersions: Map<string, string> = new Map();

  constructor(
    protected k8sApi: CustomObjectsApi,
    protected cloudflare: Cloudflare,
    protected group: string,
    protected version: string,
    protected plural: string,
    protected kind: string
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

    const listPath = `/apis/${this.group}/${this.version}/${this.plural}`;

    logger.info(`Starting watch on ${listPath}`);

    try {
      await this.watcher.watch(
        listPath,
        {},
        async (type, resource: BaseResource<TSpec, TStatus>) => {
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

  private async handleEvent(type: string, resource: BaseResource<TSpec, TStatus>) {
    const { metadata, status } = resource;
    const resourceKey = `${metadata.namespace}/${metadata.name}`;
    const resourceVersion = metadata.resourceVersion || '';

    // Skip redundant MODIFIED events with same resourceVersion
    if (type === 'MODIFIED') {
      const lastVersion = this.lastResourceVersions.get(resourceKey);
      if (lastVersion === resourceVersion) {
        logger.debug(`Skipping redundant event for ${this.kind} (resourceVersion ${resourceVersion})`);
        return;
      }

      // Skip status-only updates: only reconcile if spec changed (generation increased)
      const generation = metadata.generation || 0;
      const observedGeneration = status?.observedGeneration || 0;
      
      if (generation > 0 && generation <= observedGeneration) {
        logger.debug(`Skipping status-only update for ${this.kind} (generation ${generation} <= observed ${observedGeneration})`);
        return;
      }
      
      if (generation > observedGeneration) {
        logger.info(`Spec changed for ${this.kind} (generation ${generation} > observed ${observedGeneration})`);
      }
    }

    // Update last seen resourceVersion
    this.lastResourceVersions.set(resourceKey, resourceVersion);

    logger.info(`Event ${type} for ${this.kind} ${metadata.namespace}/${metadata.name} (rv: ${resourceVersion}, gen: ${metadata.generation})`);

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
      } as TStatus);
    }
  }

  protected async updateStatus(
    resource: BaseResource<TSpec, TStatus>,
    status: TStatus
  ) {
    const { metadata } = resource;

    try {
      await this.k8sApi.patchNamespacedCustomObjectStatus(
        this.group,
        this.version,
        metadata.namespace,
        this.plural,
        metadata.name,
        { status },
        undefined,
        undefined,
        undefined,
        { headers: { 'Content-Type': 'application/merge-patch+json' } }
      );
      logger.debug(`Status updated for ${this.kind} ${metadata.namespace}/${metadata.name}`);
    } catch (error) {
      logger.error(`Failed to update status for ${this.kind}:`, error);
    }
  }

  // Abstract methods to be implemented by subclasses
  protected abstract reconcile(resource: BaseResource<TSpec, TStatus>): Promise<void>;
  protected abstract delete(resource: BaseResource<TSpec, TStatus>): Promise<void>;

  stop() {
    if (this.abortController) {
      this.abortController.abort();
    }
  }
}
