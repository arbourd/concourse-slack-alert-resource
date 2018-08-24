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
* `channel`: *Optional*. Target channel where messages are posted. If unset the default channel of the webhook is used.
* `concourse_url`: *Optional.* The external URL that points to Concourse. Defaults to the env variable `ATC_EXTERNAL_URL`.
* `username`: *Optional.* Concourse local user (or basic auth) username. Required for non-public pipelines if using alert type `fixed` or `broke`
* `password`: *Optional.* Concourse local user (or basic auth) password. Required for non-public pipelines if using alert type `fixed` or `broke`
* `disable`: *Optional.* Disables the resource (does not send notifications). Defaults to `false`.

## Behavior

### `check`: No operation.

### `in`: No operation.

### `out`: Send a message to Slack.

Sends a structured message to Slack based on the alert type.

#### Parameters

- `alert_type`: *Optional.* The type of alert to send to Slack. See [Alert Types](#alert-types). Defaults to `default`.
- `channel`: *Optional.* Channel where this message is posted. Defaults to the `channel` setting in Source.
- `message`: *Optional.* The status message at the top of the alert. Defaults to name of alert type.
- `color`: *Optional.* The color of the notification bar as a hexadecimal. Defaults to the icon color of the alert type.
- `disable`: *Optional.* Disables the alert. Defaults to `false`.

#### Alert Types

- `default`

  <img src="./img/default.png" width="50%">

- `success`

  <img src="./img/success.png" width="50%">

- `failed`

  <img src="./img/failed.png" width="50%">

- `started`

  <img src="./img/started.png" width="50%">

- `aborted`

  <img src="./img/aborted.png" width="50%">

- `fixed`

  Fixed is a special alert type that only alerts if the previous build did not succeed. Fixed requires `username` and `password` to be set for the resource if the pipeline is not public.

  <img src="./img/fixed.png" width="50%">

- `broke`

  Broke is a special alert type that only alerts if the previous build succeed. Broke requires `username` and `password` to be set for the resource if the pipeline is not public.

  <img src="./img/broke.png" width="50%">

## Examples

### Out

Using the default alert type with custom message and color:

```yaml
resources:
- name: notify
  type: slack-alert
  source:
    url: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX

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
    url: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX

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
    url: https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
    # `alert_type: fixed` requires Concourse credentials if pipeline is private
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
