# API Reference

## Packages
- [backups.anynines.com/v1beta3](#backupsanyninescomv1beta3)

## backups.anynines.com/v1beta3

Package v1beta3 contains API Schema definitions for the backups v1beta3 API group

### Resource Types
- [BackupPolicy](#backuppolicy)
- [BackupPolicyList](#backuppolicylist)
- [Backup](#backup)
- [BackupList](#backuplist)
- [Restore](#restore)
- [RestoreList](#restorelist)

#### BackupPolicy

BackupPolicy is the Schema for the backupPolicy API

_Appears in:_
- [BackupPolicyList](#backuppoilicylist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `Backup`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata` |
| `spec` _[BackupPolicySpec](#backuppolicyspec)_ |  |
| `status` _[BackupPolicyStatus](#backuppolicystatus)_ |  |

#### BackupPolicyList

BackupPolicyList contains a list of BackupPolicies.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `BackupList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[BackupPolicy](#backuppolicy) array_ |  |

#### BackupPolicySpec

BackupPolicySpec defines the desired state of BackupPolicy.

_Appears in:_
- [BackupPolicy](#backuppolicy)

| Field | Description |
| --- | --- |
| `enabled` _bool_ | Whether the policy enabled or not, set to false instead of delete |
| `scheduleConfig` _[ScheduleConfiguration](#scheduleconfiguration)_ | ScheduleConfiguration the scheduling configuration for backups |
| `retentionConfig` _[RetentionConfiguration](#retentionconfiguration)_ | RetentionConfiguration defines the policy for retaining backups |
| `serviceInstance` _[ServiceInstanceRef](#serviceinstanceref)_ | ServiceInstance identifies the Data Service Instance to backup |
| `backupOptions` _[BackupOptions](#backupoptions)_ | BackupOptions defines the configuration for the backup process |

#### BackupPolicyStatus

BackupPolicyStatus defines the observed state of BackupPolicy.

_Appears in:_
- [BackupPolicy](#backuppolicy)

| Field | Description |
| --- | --- |
| `enabled` _bool_ | Whether the policy enabled or not, set to false instead of delete |
| `lastBackupDate` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastBackupDate is the last date when the backup process was run. |
| `nextScheduledBackup` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | NextScheduledBackup is the next date sheduled to run the backup. |
| `backupsCount` _integer_ | BackupsCount is the number of retained backups |
| `latestBackup` _[LatestBackup](#latestbackup)_ | LatestBackup is the last backup created by the policy |

#### Backup

Backup is the Schema for the backups API

_Appears in:_
- [BackupList](#backuplist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `Backup`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[BackupSpec](#backupspec)_ |  |
| `status` _[BackupStatus](#backupstatus)_ |  |

#### BackupList

BackupList contains a list of Backup.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `BackupList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
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
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Backup can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "UploadedToS3", "Terminating". |
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
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
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RestoreSpec](#restorespec)_ |  |
| `status` _[RestoreStatus](#restorestatus)_ |  |

#### RestoreList

RestoreList contains a list of Restore.

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `backups.anynines.com/v1beta3`
| `kind` _string_ | `RestoreList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
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
| `lastObservationTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#time-v1-meta)_ | LastObservationTime is the timestamp of the last time the Condition was observed to be true. |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#condition-v1-meta) array_ | Conditions include a set of not mutually exclusive states the Restore can be in, as well as the last observed time stamp for these conditions. They include "Ready", "InProgress", "Terminating". |
| `podToPoll` _[PodRef](#podref)_ | The Pod to poll to learn the status of the Restore, if the restore is in Progress. |
| `restoreID` _string_ | RestoreID is the ID of the Restore; clients can use this to poll the status of the Restore at the Pod identified by `PodToHit`. |

#### ServiceInstanceRef

ServiceInstanceRef references a Data Service Instance to backup or restore. The referenced Data Service Instance is always assumed to be in the same Kubernetes API namespace as the parent Backup/Restore API object, so there's no namespace field; we might reconsider this assumption in the future.

_Appears in:_
- [BackupPolicySpec](#backuppolicyspec)
- [BackupSpec](#backupspec)
- [RestoreSpec](#restorespec)

| Field | Description |
| --- | --- |
| `name` _string_ | Name is the name of the Kubernetes API resource that represents the Data Service Instance to backup or restore. |
| `kind` _string_ | Kind is the kind of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. Postgresql, Redis, etc...). |
| `apiGroup` _string_ | APIGroup is the API group of the Kubernetes API resource that represents the Data Service Instance to backup or restore (e.g. postgresql.anynines.com, redis.anynines.com, etc...). |

#### ScheduleConfiguration

ScheduleConfiguration defines the schedule for when the buckups should be created. it supports multiple scheduling types to provide flexibility in defining back schedules.

_Appears in:_
- [BackupPolicySpec](#backuppolicyspec)

| Field | Description |
| --- | --- |
| `type` _string_ | ScheduleType specifies the type of schedule to define date/time of next backup to run. Currently we support Cron; more types to be implemented later |
| `timeZone` _string_ | Timezone to calculate next schedule date |
| `cron` _[Cron](#cron)_ | Cron is configuration for defining expressions for scheduling backups |

#### RetentionConfiguration
RetentionConfiguration defines the policy for retaining and deleting old backups. It ensures that backups are kept for a specified number of backups or specified number of days.

_Appears in:_
- [BackupPolicySpec](#backuppolicyspec)

| Field | Description |
| --- | --- |
| `count` _integer_ | Count specifies the maximum number of backups to retain. If more backups exist than this count, the oldest backups will be deleted first |
| `period` _[RetentionPeriod](#retentionperiod)_ | Period specifies the maximum age of backups to retain e.g. 7 (days). Backups older than this period will be deleted. This period is calculated based on the creation timestamp of the backup |

#### BackupOptions
BackupOptions defines the configuration settings for backup process. This includes for now MaxRetries, more features to be added in future.

_Appears in:_
- [BackupPolicySpec](#backuppolicyspec)

| Field | Description |
| --- | --- |
| `maxRetries` _string_ | How many times the backup will be retried before aborting. Allowed values: any positive integer, or "Infinite" |

#### Cron
Cron defines the configuration settings for scheduling backups using a cron expression.

_Appears in:_
- [ScheduleConfiguration](#scheduleconfiguration)

| Field | Description |
| --- | --- |
| `Expression` _string_ | Expression is the cron expression for scheduling backups |

#### LatestBackup
LatestBackup contains the details of the last backup scheduled

_Appears in:_
- [BackupPolicyStatus](#backuppolicystatus)

| Field | Description |
| --- | --- |
| `name` _string_ | Name of the backup created |
| `status` _string_ | Status of the last backup, derived from backup conditions |

#### RetentionPeriod
RetentionPeriod specifies the maximum age of backups to retain e.g. 7 (days). Backups older than this period will be deleted. This period is calculated based on the creation timestamp of the backup. It supports multi duration types: hours, days, months and years.
_Appears in:_
- [RetentionConfiguration](#retentionconfiguration)

| Field | Description |
| --- | --- |
| `count` _integer_ | Count is the number of time unit for the retention period  e.g. 7 (days) |
| `type` _string_ | Type is the type of time unit for the retention period (hours, days, months, years) |
