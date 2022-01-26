# API Reference
## Packages

- [postgresql.anynines.com/v1alpha1](#postgresqlanyninescomv1alpha1)
- [backups.anynines.com/v1alpha1](#backupsanyninescomv1alpha1)
- [servicebindings.anynines.com/v1alpha1](#servicebindingsanyninescomv1alpha1)

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
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[PostgresqlSpec](#postgresqlspec)_ |  |

#### PostgresqlList

PostgresqlList contains a list of Postgresql

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `postgresql.anynines.com/v1alpha1`
| `kind` _string_ | `PostgresqlList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Postgresql](#postgresql)_ |  |

#### PostgresqlSpec

PostgresqlSpec defines the desired state of Postgresql

_Appears in:_
- [Postgresql](#postgresql)

| Field | Description |
| --- | --- |
| `replicas` _integer_ | Replicas is the number of replicas of the data service in the cluster. Replicas of the PostgreSQL resource will constitute a streaming replication cluster. This value should be an odd number (with the exception of the value 0) to ensure the resultant cluster can establish quorum. Only scaling up is supported and not scaling down of replicas. |
| `image` _string_ | Image is the name of the container image to use for the data service containers. |
| `backupAgentImage` _string_ | BackupAgentImage is the name of the container image to use for the backup agent sidecar container. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ | Resources is the desired compute resource requirements of PostgreSQL container within a pod in the cluster. Updating resources causes the replicas of the PostgreSQL cluster to be killed and recreated one at a time, which could potentially lead to downtime if something goes wrong during the update. |
| `postgresConfiguration` _[PostgresConfiguration](#postgresconfiguration)_ |  |

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

NamespacedName represents a Kubernetes API namespace and name. It's factored out to its own type for
reusability.

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
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ServiceBindingSpec](#servicebindingspec)_ |  |

#### ServiceBindingList

ServiceBindingList contains a list of ServiceBinding

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1alpha1`
| `kind` _string_ | `ServiceBindingList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ServiceBinding](#servicebinding)_ |  |

#### ServiceBindingSpec

ServiceBindingSpec defines the desired state of the ServiceBinding

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `instance` _[InstanceRef](#instanceref)_ | Instance identifies the Data Service Instance that the ServiceBinding binds to. |

## backups.anynines.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the backups v1alpha1 API group

### Resource Types
- [Backup](#backup)
- [BackupList](#backuplist)
- [Recovery](#recovery)
- [RecoveryList](#recoverylist)

#### Backup

Backup is the Schema for the backups API

_Appears in:_
- [BackupList](#backuplist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `Backup`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[BackupSpec](#backupspec)_ |  |

#### BackupList

BackupList contains a list of Backup

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `BackupList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Backup](#backup)_ |  |

#### BackupSpec

BackupSpec defines the desired state of Backup.

_Appears in:_
- [Backup](#backup)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to backup. |

#### PodRef

PodRef describes a reference to a Pod.

_Appears in:_
- [RecoveryStatus](#recoverystatus)

| Field | Description |
| --- | --- |
| `namespacedName` _string_ | NamespacedName is the namespaced name of the Pod. |
| `uid` _UID_ | UID is the UID of the Pod. |
| `ip` _string_ | IP is the IP of the Pod. |

#### Recovery

Recovery is the Schema for the recoveries API

_Appears in:_
- [RecoveryList](#recoverylist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `Recovery`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RecoverySpec](#recoveryspec)_ |  |

#### RecoveryList

RecoveryList contains a list of Recovery

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1alpha1`
| `kind` _string_ | `RecoveryList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Recovery](#recovery)_ |  |

#### RecoverySpec

RecoverySpec defines the desired state of Recovery

_Appears in:_
- [Recovery](#recovery)

| Field | Description |
| --- | --- |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to restore. |
| `backupName` _string_ | BackupName is the name of the Backup API object to use for the Recovery; the namespace is assumed to be the same as the one for the Recovery object, we might reconsider this assumption in the future. |

#### ServiceInstanceRef

ServiceInstanceRef references a Data Service Instance to backup or restore. The referenced Data
Service Instance is always assumed to be in the same Kubernetes API namespace as the parent
Backup/Recovery API object, so there's no namespace field; we might reconsider this assumption inthe future. The API Group to which the kind belongs must be inferred by implementers of the API.

_Appears in:_
- [BackupSpec](#backupspec)
- [RecoverySpec](#recoveryspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the Kubernetes API resource that represents the Data Service Instance to backup or restore. |
| `kind` _string_ | Kind is the kind of the Kubernetes API resource that represents the Data Service Instance to to backup or restore (e.g. PostgreSQL, Redis, etc...). |

#### StatusCondition

StatusCondition is a summary of the last observed status of a Backup or a Recovery operation. TODO: Find a better name for this type.

_Appears in:_
- [BackupStatus](#backupstatus)
- [RecoveryStatus](#recoverystatus)

| Field | Description |
| --- | --- |
| `type` _[StatusConditionType](#statusconditiontype)_ | Type is a short camelCase message describing the last observed condition of the backup or recovery. Can be one of `succeeded`, `inProgress`, `failed`. |
| `error` _string_ | Error contains a message describing what went wrong in case `Type` has value `failed`. |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |

#### StatusConditionType

_Underlying type:_ `string`

_Appears in:_
- [StatusCondition](#statuscondition)