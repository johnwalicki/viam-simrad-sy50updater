# Module viam-simrad-sy50updater

Prompt the user to upgrade the version of Simrad SY50 software installed on their Windows 11 system.

Display a Message box on Win11 asking the user if they want to upgrade Simrad SY50 - Yes / No / Cancel.

If the user clicks Yes, invoke the generic Viam Windows Updater (dependency) do_command() to perform
the download, install.

## Model walicki:sy50updater:SY50-updater

Provide a description of the model and any relevant information.

### Configuration
The following attribute template can be used to configure this model:

```json
{
  "prompt": <bool>,
  "version_target": <string>,
  "download_url": <string>
}
```

#### Attributes

The following attributes are available for this model:

| Name             | Type   | Inclusion | Description                                            |
|------------------|--------|-----------|--------------------------------------------------------|
| `version_target` | string | Required  | Target version of Simrad SY50 to upgrade the system to |
| `download_url`   | string | Required  | Download URL of the Simrad SY50 installer              |
| `prompt`         | bool   | Optional  | Whether to prompt the user to install a new version    |

#### Example Configuration

```json
{
  "prompt": false,
  "version_target": "24.7.9230.28743",
  "download_url": "https://www.simrad.club/sy50/sy50_setup_24_7_3.zip"
}
```

### DoCommand

This module implements DoCommand, but the parameters are defined above. The `prompt` and `download_url` could move here.

#### Example DoCommand

```json
{
  "command_name": {
    "arg1": "foo",
    "arg2": 1
  }
}
```
