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
"attribute_1": <float>,
"attribute_2": <string>
}
```

#### Attributes

The following attributes are available for this model:

| Name          | Type   | Inclusion | Description                |
|---------------|--------|-----------|----------------------------|
| `attribute_1` | float  | Required  | Description of attribute 1 |
| `attribute_2` | string | Optional  | Description of attribute 2 |

#### Example Configuration

```json
{
  "attribute_1": 1.0,
  "attribute_2": "foo"
}
```

### DoCommand

If your model implements DoCommand, provide an example payload of each command that is supported and the arguments that can be used. If your model does not implement DoCommand, remove this section.

#### Example DoCommand

```json
{
  "command_name": {
    "arg1": "foo",
    "arg2": 1
  }
}
```
