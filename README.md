# concourse-slack-alert-resource

A structured and opinionated Slack notification resource for [Concourse](https://concourse.ci/).

## Installing

Use this resource by adding the following to the resource_types section of a pipeline config:

```yaml
resource_types:

- name: slack-alert
  type: docker-image
  source:
    repository: arbourd/concourse-slack-alert-resource
```

See the [Concourse docs](https://concourse-ci.org/resource-types.html) for more details on adding `resource_types` to a pipeline config.

## Source Configuration

* `url`: *Required.* Slack webhook URL.
* `username`: *Optional.* Concourse basic auth username. Required if using `alert_type: fixed`
* `password`: *Optional.* Concourse basic auth password. Required if using `alert_type: fixed`

## Behavior

### `check`: No operation.

### `in`: No operation.

### `out`: Send a message to Slack.

Sends a structured message to Slack based on the alert type.

#### Parameters

* `alert_type`: *Required.* The type of alert to send to Slack. There are 4 options: `success`, `failed`, `aborted` and `fixed`.

## Examples

### Out

Using build hooks with equivilent alert types:

```yaml
resources:
- name: notify
  type: slack-alert
  source:
    url: https://hooks.slack.com/services/ANER808F/SKDCVS3B/vvPBAWQVPHDKejdeThDiE4wrg

jobs:
  # ...
  plan:
  - put: some-other-task
    on_success:
      put: notify
      params:
        alert_type: success
    on_failure:
      put: notify
      params:
        alert_type: failed
    on_abort:
      put: notify
      params:
        alert_type: aborted
```

Using the `fixed` alert type:

```yaml
resources:
- name: notify
  type: slack-alert
  source:
    url: https://hooks.slack.com/services/ANER808F/SKDCVS3B/vvPBAWQVPHDKejdeThDiE4wrg

    # alert_type: fixed requires Concourse credentials to check previous builds
    username: concourse
    password: concourse

jobs:
  # ...
  plan:
  - put: some-other-task
    on_success:
      put: notify
      params:
        # will only alert if build was successful and fixed
        alert_type: fixed
```
