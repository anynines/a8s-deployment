# API Reference
## Packages

- [postgresql.anynines.com/v1alpha1](#postgresqlanyninescomv1alpha1)
- [servicebindings.anynines.com/v1alpha1](#servicebindingsanyninescomv1alpha1)
- [backups.anynines.com/v1alpha1](#backupsanyninescomv1alpha1)

## postgresql.anynines.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the postgresql v1alpha1 API group

### Resource Types
- [Postgresql](#postgresql)
- [PostgresqlList](#postgresqllist)

#### PostgresConfiguration

_Appears in:_
- [PostgresqlSpec](#postgresqlspec)

| Field | Description |
| --- | --- |
| `maxConnections` _integer_ | MaxConnections determines the maximum number of concurrent connections to the database server. Updating MaxConnections will trigger a restart of the PostgreSQL instance. |
| `sharedBuffers` _integer_ | SharedBuffers sets the amount of memory (usually in 8KB) the database server uses for shared memory buffers. If this value is specified without units, it is taken as blocks, that is BLCKSZ bytes, typically 8kB. This setting must be at least 128 kilobytes. However, settings significantly higher than the minimum are usually needed for good performance. Updating SharedBuffers will trigger a restart of the PostgreSQL instance. |
| `maxReplicationSlots` _integer_ | MaxReplicationSlots specifies the maximum number of replication slots that the server can support. Updating MaxReplicationSlots will trigger a restart of the PostgreSQL instance. |
| `maxWALSenders` _integer_ | MaxWALSenders specifies the maximum number of concurrent connections from standby servers or streaming base backup clients (i.e., the maximum number of simultaneously running WAL sender processes). The value 0 means replication is disabled. Abrupt disconnection of a streaming client might leave an orphaned connection slot behind until a timeout is reached, so this parameter should be set slightly higher than the maximum number of expected clients so disconnected clients can immediately reconnect. Updating MaxWALSenders will trigger a restart of the PostgreSQL instance. |
| `maxLocksPerTransaction` _integer_ | MaxLocksPerTransaction sets the maximum number of locks per transaction. The shared lock table tracks locks on max_locks_per_transaction * (max_connections + max_prepared_transactions) objects (e.g., tables); hence, no more than this many distinct objects can be locked at any one time. This parameter controls the average number of object locks allocated for each transaction; individual transactions can lock more objects as long as the locks of all transactions fit in the lock table. This is not the number of rows that can be locked; that value is unlimited. The default, 64, has historically proven sufficient, but you might need to raise this value if you have queries that touch many different tables in a single transaction, e.g., query of a parent table with many children. Updating MaxLocksPerTransaction will trigger a restart of the PostgreSQL instance. |
| `statementTimeoutMillis` _integer_ | StatementTimeoutMillis is the timeout in milliseconds after which any statement that takes more than the specified number is aborted. The counter is started from the time the command arrives at the server from the client. If LogMinErrorStatement statement is set to ERROR or lower, the statement that timed out will also be logged. A value of zero (the default) turns this off. |
| `sslCiphers` _string_ | SSLCiphers specifies the allowed SSL ciphers (https://www.postgresql.org/docs/13/runtime-config-connection.html#GUC-SSL-CIPHERS) |
| `sslMinProtocolVersion` _string_ | SSLMinProtocolVersion sets the minimum SSL/TLS protocol version to use |
| `tempFileLimitKiloBytes` _integer_ | TempFileLimitKiloBytes specifies the maximum amount of disk space that a process can use for temporary files, such as sort and hash temporary files, or the storage file for a held cursor. |
| `walWriterDelayMillis` _integer_ | WALWriterDelayMillis specifies the time (in milliseconds) between WAL flushed performed in the WAL writer. After flushing WAL the writer sleeps for the length of time given by WALWriterDelayMillis, unless woken up sooner by an asynchronously committing transaction. |
| `synchronousCommit` _string_ | SynchronousCommit specifies whether transaction commit will wait for WAL records to be written to disk before the command returns a success indication to the client. |
| `trackIOTiming` _string_ | TrackIOTiming enables timing of database I/O calls. This parameter is off by default, because it will repeatedly query the operating system for the current time, which may cause significant overhead on some platforms. |
| `archiveTimeoutSeconds` _integer_ | ArchiveTimeoutSeconds is the timeout in seconds which defines the limit how old unarchived data can be, you can set ArchiveTimeoutSeconds to force the server to switch to a new WAL segment file periodically. When this parameter is greater than zero, the server will switch to a new segment file whenever this amount of time has elapsed since the last segment file switch. |
| `clientMinMessages` _string_ | ClientMinMessages specifies which message levels are sent to the client. |
| `logMinMessages` _string_ | LogMinMessages controls which message levels are written to the server log. |
| `logMinErrorStatement` _string_ | LogMinErrorStatement controls which SQL statements that cause an error condition are recorded in the server log. The current SQL statement is included in the log entry for any message of the specified severity or higher. |
| `logStatement` _string_ | LogStatement controls which SQL statements are logged. |
| `logErrorVerbosity` _string_ | LogErrorVerbosity controls the amount of detail written in the server log for each message that is logged. |

#### Postgresql

Postgresql is the Schema for the postgresqls API

_Appears in:_
- [PostgresqlList](#postgresqllist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `postgresql.anynines.com/v1alpha1`
| `kind` _string_ | `Postgresql`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[PostgresqlSpec](#postgresqlspec)_ |  |
| `status` _[PostgresqlStatus](#postgresqlstatus)_ |  |

#### PostgresqlList

PostgresqlList contains a list of Postgresql

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `postgresql.anynines.com/v1alpha1`
| `kind` _string_ | `PostgresqlList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Postgresql](#postgresql) array_ |  |

#### PostgresqlSchedulingConstraints

_Appears in:_
- [PostgresqlSpec](#postgresqlspec)

| Field | Description |
| --- | --- |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#toleration-v1-core) array_ | Tolerations is the list of tolerations that the Pods of the PostgreSQL instance will have with respect to the taints of the Kubernetes cluster nodes. It can be used to affect scheduling of the Pods of the PostgreSQL instance on the Kubernetes cluster nodes. More information at https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ and https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#toleration-v1-core . If you don't know what are the specific taints on the nodes of the Kubernetes cluster you're using, you should ask your cluster administrator. Updating this field will result in re-creation and re-scheduling of all the Pods of the PostgreSQL instance, so *there may be downtime*. |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#affinity-v1-core)_ | Affinity groups the fields that govern node affinity, Pod affinity, and Pod anti affinity. It has exactly the same syntax and semantics of the built-in Kubernetes affinity type: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core . It is copied as it is in the spec of the Pods of the PostgreSQL instance, and can be used to attract said Pods to specific Kubernetes cluster nodes based on those nodes' labels and on the labels of other Pods running on the nodes. In the latter case repulsion rules between Pods can also be configured via the PodAntiAffinity field. An important use case for repulsion rules would be HA: it can be ensured that the replicas of the same PostgreSQL instance repel each other and end up on different nodes or even availability zones. More information at https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity . To properly configure this field you might need to know which labels the DSI pods get. That's the union of the DSI's `metadata.labels`; the a8s-reserved labels "a8s.a9s/dsi-name": "<name of the instance>", "a8s.a9s/dsi-group": "postgresql.anynines.com", "a8s.a9s/dsi-kind": "Postgresql"; the label "a8s.a9s/replication-role", set to the value "master" for the primary pod and "replica" for the standby pods; the standard labels that Kubernetes adds to all pods managed via a statefulset. Updating this field will result in re-creation and re-scheduling of all the Pods of the PostgreSQL instance, so *there may be downtime*. |

#### PostgresqlSpec

PostgresqlSpec defines the desired state of Postgresql

_Appears in:_
- [Postgresql](#postgresql)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | Replicas is the number of replicas of the data service in the cluster. Replicas of the PostgreSQL resource will constitute a streaming replication cluster. This value should be an odd number (with the exception of the value 0) to ensure the resultant cluster can establish quorum. Only scaling up is supported and not scaling down of replicas. |
| `version` _integer_ | Version specifies the PostgreSQL version that the instance should use. It is required to pick a version and it cannot be changed afterwards, since major version upgrades are  currently unsupported. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#resourcerequirements-v1-core)_ | Resources is the desired compute resource requirements of PostgreSQL container within a pod in the cluster. Updating resources causes the replicas of the PostgreSQL cluster to be killed and recreated one at a time, which could potentially lead to downtime if something goes wrong during the update. |
| `volumeSize` _Quantity_ | VolumeSize sets the size of the persistent volume of the PostgreSQL instance, the minimum size is 0.5Gi. The size is to be specified as a plain integer or as a fixed-point number using one of these suffixes: E, P, T, G, M, K i bim 1 test, corresponding bla to kilo-, mega-, gigabytes, etc. You can also use the power-of-two equivalents: Ei, Pi, Ti, Gi, Mi, Ki, corresponding to kibi-, mebi-, gibibytes, etc. For example  a value of "0.5Gi" corresponds to an instance with a persistent volume of 0.5 gibibytes. |
| `postgresConfiguration` _[PostgresConfiguration](#postgresconfiguration)_ |  |
| `schedulingConstraints` _[PostgresqlSchedulingConstraints](#postgresqlschedulingconstraints)_ | SchedulingConstraints contains subfields that affect how the Pods of the Postgresql instance will be scheduled onto Kubernetes cluster nodes. The subfields map directly to Kubernetes API primitives such as node taints, tolerations, affinity and (anti)affinity. See the documentation of each subfield for more details. Together, the subfields of SchedulingConstraints allow you to express constraints such as "Pods of this Postgresql instance MUST be scheduled to different availability zones", or "Pods of this Postgresql instance SHOULD preferrably (but not mandatorily) be scheduled to nodes that have a SSD", and many more. As a warning, the subfields of SchedulingConstraints can interfere with each other, so when you set one of them you should consider how it will interact with the values that you set for other subfields. |
| `extensions` _string array_ | Extensions defines a list of PostgreSQL extensions which should be installed. Installing means that the binaries and libraries of the defined extensions are moved to the PostgreSQL extension directory. The extensions are NOT loaded by default (i.e. by using the PostgreSQL "CREATE EXTENSION" command). Updating the list of extensions will cause a rolling update of the PostgreSQL instance. |

#### PostgresqlStatus

PostgresqlStatus defines the observed state of Postgresql

_Appears in:_
- [Postgresql](#postgresql)

| Field | Description |
| --- | --- |
| `readyReplicas` _integer_ |  |
| `clusterStatus` _string_ | ClusterStatus is a summary of the current status of the cluster. Careful, if ReadyReplicas < 'spec.Replicas' or `spec.Replicas` == 0 the status equals "NotRunning". |
| `error` _string_ |  |

## servicebindings.anynines.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the servicebindings v1alpha1 API group

### Resource Types
- [ServiceBinding](#servicebinding)
- [ServiceBindingList](#servicebindinglist)

#### InstanceRef

InstanceRef is a reference to a Data Service Instance.

_Appears in:_
- [ServiceBindingSpec](#servicebindingspec)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | APIVersion is the <api_group>/<version> of the referenced Data Service Instance, e.g. "postgresql.anynines.com/v1alpha1" or "redis.anynines.com/v1alpha1". |
| `kind` _string_ | Kind is the Kubernetes API Kind of the referenced Data Service Instance. |
| `NamespacedName` _[NamespacedName](#namespacedname)_ | NamespacedName is the Kubernetes API Kind of the referenced Data Service Instance. |

#### NamespacedName

NamespacedName represents a Kubernetes API namespace and name. It's factored out to its own type for reusability.

_Appears in:_
- [InstanceRef](#instanceref)
- [ServiceBindingStatus](#servicebindingstatus)

| Field | Description |
| --- | --- |
| `namespace` _string_ |  |
| `name` _string_ |  |

#### ServiceBinding

ServiceBinding is the Schema for the servicebindings API

_Appears in:_
- [ServiceBindingList](#servicebindinglist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1alpha1`
| `kind` _string_ | `ServiceBinding`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ServiceBindingSpec](#servicebindingspec)_ |  |
| `status` _[ServiceBindingStatus](#servicebindingstatus)_ |  |

#### ServiceBindingList

ServiceBindingList contains a list of ServiceBinding

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1alpha1`
| `kind` _string_ | `ServiceBindingList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ServiceBinding](#servicebinding) array_ |  |

#### ServiceBindingSpec

ServiceBindingSpec defines the desired state of the ServiceBinding

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `instance` _[InstanceRef](#instanceref)_ | Instance identifies the Data Service Instance that the ServiceBinding binds to. |

#### ServiceBindingStatus

ServiceBindingStatus defines the observed state of the ServiceBinding

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `secret` _[NamespacedName](#namespacedname)_ | Secret contains the namespace and name of the Kubernetes API secret that stores the credentials and information (e.g. URL) associated to the service binding to access the bound Data Service Instance. |
| `implemented` _boolean_ | Implemented is `true` if and only if the service binding has been implemented by creating a user with the appropriate permissions in the bound Data Service Instance. Users can safely consume the service binding secret identified by `Secret` IF AND ONLY IF `Implemented` is true. In other words, even if the secret identified by `Secret` gets created before `Implemented` becomes true, users MUST NOT consume that secret before `Implemented` has become true. |
| `error` _string_ | Error is a message explaining why the service binding could not be implemented if that's the case. |

## servicebindings.anynines.com/v1beta3

Package v1beta3 contains API Schema definitions for the servicebindings v1beta3 API group

### Resource Types
- [ServiceBinding](#servicebinding)
- [ServiceBindingList](#servicebindinglist)

#### InstanceRef

InstanceRef is a reference to a Data Service Instance.

_Appears in:_
- [ServiceBindingSpec](#servicebindingspec)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | APIVersion is the <api_group>/<version> of the referenced Data Service Instance, e.g. "postgresql.anynines.com/v1beta3" or "redis.anynines.com/v1alpha1". |
| `kind` _string_ | Kind is the Kubernetes API Kind of the referenced Data Service Instance. |
| `NamespacedName` _[NamespacedName](#namespacedname)_ | NamespacedName is the Kubernetes API Kind of the referenced Data Service Instance. |

#### NamespacedName

NamespacedName represents a Kubernetes API namespace and name. It's factored out to its own type for reusability.

_Appears in:_
- [InstanceRef](#instanceref)
- [ServiceBindingStatus](#servicebindingstatus)

| Field | Description |
| --- | --- |
| `namespace` _string_ |  |
| `name` _string_ |  |

#### ServiceBinding

ServiceBinding is the Schema for the servicebindings API

_Appears in:_
- [ServiceBindingList](#servicebindinglist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1beta3`
| `kind` _string_ | `ServiceBinding`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ServiceBindingSpec](#servicebindingspec)_ |  |
| `status` _[ServiceBindingStatus](#servicebindingstatus)_ |  |

#### ServiceBindingList

ServiceBindingList contains a list of ServiceBinding.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1beta3`
| `kind` _string_ | `ServiceBindingList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ServiceBinding](#servicebinding) array_ |  |

#### ServiceBindingSpec

ServiceBindingSpec defines the desired state of the ServiceBinding.

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `instance` _[InstanceRef](#instanceref)_ | Instance identifies the Data Service Instance that the ServiceBinding binds to. |

#### ServiceBindingStatus

ServiceBindingStatus defines the observed state of the ServiceBinding.

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `secret` _[NamespacedName](#namespacedname)_ | Secret contains the namespace and name of the Kubernetes API secret that stores the credentials and information (e.g. URL) associated to the service binding to access the bound Data Service Instance. |
| `implemented` _boolean_ | Implemented is `true` if and only if the service binding has been implemented by creating a user with the appropriate permissions in the bound Data Service Instance. Users can safely consume the service binding secret identified by `Secret` IF AND ONLY IF `Implemented` is true. In other words, even if the secret identified by `Secret` gets created before `Implemented` becomes true, users MUST NOT consume that secret before `Implemented` has become true. |
| `error` _string_ | Error is a message explaining why the service binding could not be implemented if that's the case. |

## backups.anynines.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the backups v1alpha1 API group

### Resource Types
- [Backup](#backup)
- [BackupList](#backuplist)
- [Restore](#restore)
- [RestoreList](#restorelist)

#### Backup

Backup is the Schema for the backups API

_Appears in:_
- [BackupList](#backuplist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `Backup`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[BackupSpec](#backupspec)_ |  |
| `status` _[BackupStatus](#backupstatus)_ |  |

#### BackupList

BackupList contains a list of Backup

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `BackupList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Backup](#backup) array_ |  |

#### BackupSpec

BackupSpec defines the desired state of Backup.

_Appears in:_
- [Backup](#backup)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to backup. |
| `maxRetries` _string_ | How many times the backup will be retried before aborting. Allowed values: any positive integer, or "Infinite" |

#### BackupStatus

BackupStatus defines the observed state of Backup.

_Appears in:_
- [Backup](#backup)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Backup can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "UploadedToS3", "Terminating". |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
| `retries` _integer_ | Number of times the backup has been retried |
| `podUsedNamespacedName` _string_ | PodUsedNamespacedName is the namespaced name of the DSI Pod to which the backup request was sent. TODO: Represent this jointly with `PodUsedID` (below) via a PodRef. |
| `podUsedUID` _UID_ | PodUsedUID is the UID of the DSI Pod to which the backup request was sent. TODO: Represent this jointly with `PodUsedNamespacedName` (above) via a PodRef. |
| `backupID` _string_ | BackupID is the ID of the Backup; clients can use this to poll the status of the Backup at the Pod identified by `PodUsedID`. |

#### PodRef

PodRef describes a reference to a Pod.

_Appears in:_
- [RestoreStatus](#restorestatus)

| Field | Description |
| --- | --- |
| `namespacedName` _string_ | NamespacedName is the namespaced name of the Pod. |
| `uid` _UID_ | UID is the UID of the Pod. |
| `ip` _string_ | IP is the IP of the Pod. |

#### Restore

Restore is the Schema for the restore API

_Appears in:_
- [RestoreList](#restorelist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `Restore`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RestoreSpec](#restorespec)_ |  |
| `status` _[RestoreStatus](#restorestatus)_ |  |

#### RestoreList

RestoreList contains a list of Restore.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `RestoreList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Restore](#restore) array_ |  |

#### RestoreSpec

RestoreSpec defines the desired state of Restore.

_Appears in:_
- [Restore](#restore)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to restore. |
| `backupName` _string_ | BackupName is the name of the Backup API object to use for the Restore; the namespace is assumed to be the same as the one for the Restore object, we might reconsider this assumption in the future. |

#### RestoreStatus

RestoreStatus defines the observed state of Restore.

_Appears in:_
- [Restore](#restore)

| Field | Description |
| --- | --- |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Restore can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "Terminating". |
| `podToPoll` _[PodRef](#podref)_ | The Pod to poll to learn the status of the Restore, if the restore is in Progress. |
| `restoreID` _string_ | RestoreID is the ID of the Restore; clients can use this to poll the status of the Restore at the Pod identified by `PodToHit`. |

#### ServiceInstanceRef

ServiceInstanceRef references a Data Service Instance to backup or restore. The referenced Data Service Instance is always assumed to be in the same Kubernetes API namespace as the parent Backup/Restore API object, so there's no namespace field; we might reconsider this assumption in the future.

_Appears in:_
- [BackupSpec](#backupspec)
- [RestoreSpec](#restorespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the Kubernetes API resource that represents the Data Service Instance to backup or restore. |
| `kind` _string_ | Kind is the kind of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. Postgresql, Redis, etc...). |
| `apiGroup` _string_ | APIGroup is the API group of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. postgresql.anynines.com, redis.anynines.com, etc...). |

## backups.anynines.com/v1beta3

Package v1beta3 contains API Schema definitions for the backups v1beta3 API group

### Resource Types
- [Backup](#backup)
- [BackupList](#backuplist)
- [Restore](#restore)
- [RestoreList](#restorelist)

#### Backup

Backup is the Schema for the backups API

_Appears in:_
- [BackupList](#backuplist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `Backup`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[BackupSpec](#backupspec)_ |  |
| `status` _[BackupStatus](#backupstatus)_ |  |

#### BackupList

BackupList contains a list of Backup.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `BackupList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Backup](#backup) array_ |  |

#### BackupSpec

BackupSpec defines the desired state of Backup.

_Appears in:_
- [Backup](#backup)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to backup. |
| `maxRetries` _string_ | How many times the backup will be retried before aborting. Allowed values: any positive integer, or "Infinite" |

#### BackupStatus

BackupStatus defines the observed state of Backup.

_Appears in:_
- [Backup](#backup)

| Field | Description |
| --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Backup can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "UploadedToS3", "Terminating". |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
| `retries` _integer_ | Number of times the backup has been retried |
| `podUsedNamespacedName` _string_ | PodUsedNamespacedName is the namespaced name of the DSI Pod to which the backup request was sent. TODO: Represent this jointly with `PodUsedID` (below) via a PodRef. |
| `podUsedUID` _UID_ | PodUsedUID is the UID of the DSI Pod to which the backup request was sent. TODO: Represent this jointly with `PodUsedNamespacedName` (above) via a PodRef. |
| `backupID` _string_ | BackupID is the ID of the Backup; clients can use this to poll the status of the Backup at the Pod identified by `PodUsedID`. |

#### PodRef

PodRef describes a reference to a Pod.

_Appears in:_
- [RestoreStatus](#restorestatus)

| Field | Description |
| --- | --- |
| `namespacedName` _string_ | NamespacedName is the namespaced name of the Pod. |
| `uid` _UID_ | UID is the UID of the Pod. |
| `ip` _string_ | IP is the IP of the Pod. |

#### Restore

Restore is the Schema for the restore API

_Appears in:_
- [RestoreList](#restorelist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `Restore`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RestoreSpec](#restorespec)_ |  |
| `status` _[RestoreStatus](#restorestatus)_ |  |

#### RestoreList

RestoreList contains a list of Restore.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `RestoreList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Restore](#restore) array_ |  |

#### RestoreSpec

RestoreSpec defines the desired state of Restore.

_Appears in:_
- [Restore](#restore)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to restore. |
| `backupName` _string_ | BackupName is the name of the Backup API object to use for the Restore; the namespace is assumed to be the same as the one for the Restore object, we might reconsider this assumption in the future. |

#### RestoreStatus

RestoreStatus defines the observed state of Restore.

_Appears in:_
- [Restore](#restore)

| Field | Description |
| --- | --- |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Restore can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "Terminating". |
| `podToPoll` _[PodRef](#podref)_ | The Pod to poll to learn the status of the Restore, if the restore is in Progress. |
| `restoreID` _string_ | RestoreID is the ID of the Restore; clients can use this to poll the status of the Restore at the Pod identified by `PodToHit`. |

#### ServiceInstanceRef

ServiceInstanceRef references a Data Service Instance to backup or restore. The referenced Data Service Instance is always assumed to be in the same Kubernetes API namespace as the parent Backup/Restore API object, so there's no namespace field; we might reconsider this assumption in the future.

_Appears in:_
- [BackupSpec](#backupspec)
- [RestoreSpec](#restorespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the Kubernetes API resource that represents the Data Service Instance to backup or restore. |
| `kind` _string_ | Kind is the kind of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. Postgresql, Redis, etc...). |
| `apiGroup` _string_ | APIGroup is the API group of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. postgresql.anynines.com, redis.anynines.com, etc...). |

