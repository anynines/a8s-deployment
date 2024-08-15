# Backup Policy

The `BackupPolicy` controller manages the scheduling and lifecycle of `Backup` resources in a Kubernetes cluster.
It ensures backups are created according to a defined schedule and retains or deletes old backups based on a specified retention policy.

## Features

### Scheduling

Create backups based on a configured schedule.

### Retention Policy

Allows defining retention policies based on either the number of backups to retain or the maximum age of backups.
You can retain backups for a specified number or define a maximum age for backups (e.g., 7 days).
The controller will delete Backups above this limit, starting with the oldest ones. This age is calculated based on the creation timestamp of the backup. 
It supports multiple duration types: `hour`, `day`, `month`, and `year`.

> More features will be added to the policy in future such as `Compression`, `Notifications`, `Encryption`

## Configuration

`BackupPolicy` consists of different configuration that manages the backup scheduling and other features.

### ScheduleConfiguration

The ScheduleConfiguration defines the schedule type to be used. We currently support Cron configuration, which is the most common method for creating schedules.
More types may be supported in the future based on user needs.

The configuration allows you to specify a timezone for calculating the schedule. The default is `UTC`.
The config gives the ability to specify timezone where the controller will calculate the schedule based upon, the default is `UTC`.
*Sample*

```yaml
scheduleConfig:
    scheduleType: Cron
    timezone: "CET"
    Cron:
        expression: "0 0 * * *"
```

### RetentionConfiguration

The RetentionConfiguration defines the policy for retaining and deleting old backups.
It ensures that backups are kept for a specified duration or up to a specified number of backups.

#### Retention by Count

With retention by count, you define the maximum number of backups to retain.
The controller will delete Backups above this limit, starting with the oldest ones(based on creation timestamp).

##### Retention by Count Sample

```yaml
spec:
    retentionConfig:
        count: 5
```

#### Retention by Period

When using Period for retention, you specify count and type.
Type is the duration type a backup exceeding the count specified will be deleted starting from the `creationTimestamp` i.e. 5 `days`, 2 `month` or even 3 `years`

##### Retention by Period Sample

```yaml
spec:
    retentionConfig:
        retentionPeriod:
            type: "day"
            count: 15
```

In the above example, the controller fetches the backups created by the policy and checks their creationTimestamp against a 15-day window.
If the timestamp falls outside this window (i.e., the backup is older than 15 days), it marks the backup for deletion.

## BackupPolicy Controller

Once you create the policy, the controller will create a backup immediately. The controller creates a backup if there are no backups or if the scheduled time has elapsed.

After creating a policy, you can disable it, which allows you to keep it instead of just deleting the resource.
However, updating the status and reconciliation will not be possible until it is enabled again. 
All policies created are enabled by default.

The controller will generate backup name with the convention backup-, where each backup created will be labeled with the key a8s.a9s/policy and the value as the policy name.

Currently, the controller will create backups without checking if there is a running backup. We need to validate the case of long running backups.
