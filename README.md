[![Build Status](https://travis-ci.com/arbourd/concourse-slack-alert-resource.svg?branch=master)](https://travis-ci.com/arbourd/concourse-slack-alert-resource)

# concourse-slack-alert-resource

A structured and opinionated Slack notification resource for [Concourse](https://concourse.ci/).

<img src="./img/default.png" width="60%">

The message is built by using Concourse's [resource metadata](https://concourse-ci.org/implementing-resources.html#resource-metadata) to show the pipeline, job, build number and a URL.

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
* `concourse_url`: *Optional.* The external URL that points to Concourse. Defaults to the env variable `ATC_EXTERNAL_URL`.
* `username`: *Optional.* Concourse basic auth username. Required for non-public pipelines if using alert type `fixed`
* `password`: *Optional.* Concourse basic auth password. Required for non-public pipelines if using alert type `fixed`

## Behavior

### `check`: No operation.

### `in`: No operation.

### `out`: Send a message to Slack.

Sends a structured message to Slack based on the alert type.

#### Parameters

- `alert_type`: *Optional.* The type of alert to send to Slack. Defaults to `default`.

  `default`

  <img src="./img/default.png" width="50%">

  `success`

  <img src="./img/success.png" width="50%">

  `failed`

  <img src="./img/failed.png" width="50%">

  `started`

  <img src="./img/started.png" width="50%">

  `aborted`

  <img src="./img/aborted.png" width="50%">

  `fixed`

  Fixed is a special alert type that requires both `username` and `password` to be set in Source and will only alert if the previous build was a failure.

  <img src="./img/fixed.png" width="50%">

- `message`: *Optional.* The status message at the top of the alert. Defaults to name of alert type, except for default which is nothing.
- `color`: *Optional.* The color of the notification bar as a hexadecimal. Defaults to the icon color of the alert type.
- `disable`: *Optional.* Disables the alert. Defaults to `false`.

## Examples

### Out

Using the default alert type with custom message and color:

```yaml
resources:
- name: notify
  type: slack-alert
  source:
    url: https://hooks.slack.com/services/ANER808F/SKDCVS3B/vvPBAWQVPHDKejdeThDiE4wrg

jobs:
  # ...
  plan:
  - put: notify
    params:
      message: Completed
      color: "#eeeeee"
```

Using built-in alert types with appropriate build hooks:

```yaml
resources:
- name: notify
  type: slack-alert
  source:
    url: https://hooks.slack.com/services/ANER808F/SKDCVS3B/vvPBAWQVPHDKejdeThDiE4wrg

jobs:
  # ...
  plan:
  - put: notify
    params:
      alert_type: started
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
