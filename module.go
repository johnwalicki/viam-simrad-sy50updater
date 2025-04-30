package sy50updater

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"gopkg.in/toast.v1"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/utils/rpc"
)

var (
	Sy50Updater      = resource.NewModel("walicki", "sy50updater", "SY50-updater")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(sensor.API, Sy50Updater,
		resource.Registration[sensor.Sensor, *sensorConfig]{
			Constructor: newSy50updaterSy50Updater,
		},
	)
}

type sensorConfig struct {
	DownloadURL   string `json:"download_url"`
	versionTarget string `json:"version_target"`
}

// Validate ensures all parts of the config are valid and important fields exist.
// Returns implicit dependencies based on the config.
// The path is the JSON path in your robot's config (not the `Config` struct) to the
// resource being validated; e.g. "components.0".
func (cfg *sensorConfig) Validate(path string) ([]string, error) {
	// Add config validation code here
	return nil, nil
}


type sy50updaterSy50Updater struct {
	resource.AlwaysRebuild

	name resource.Name

	logger logging.Logger
	cfg    *sensorConfig

	cancelCtx  context.Context
	cancelFunc func()
}


func newSy50updaterSy50Updater(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	conf, err := resource.NativeConfig[*sensorConfig](rawConf)
	if err != nil {
		return nil, err
	}

	return NewSy50Updater(ctx, deps, rawConf.ResourceName(), conf, logger)
}


func NewSy50Updater(ctx context.Context, deps resource.Dependencies, name resource.Name, conf *sensorConfig, logger logging.Logger) (sensor.Sensor, error) {

	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &sy50updaterSy50Updater{
		name:       name,
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}
	return s, nil
}


func (s *sy50updaterSy50Updater) Name() resource.Name {
	return s.name
}


func (s *sy50updaterSy50Updater) NewClientFromConn(ctx context.Context, conn rpc.ClientConn, remoteName string, name resource.Name, logger logging.Logger) (sensor.Sensor, error) {
	return nil, errUnimplemented
}


func (s *sy50updaterSy50Updater) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	// Find the Simrad SY50 version in the Windows Registry
	programName := "Simrad SY50"
	version, err := getWindowsProgramVersion(programName)
	if err != nil {
		// Not installed or not found
		version = "Not installed"
	}

	s.logger.Infof("Simrad SY50 version details: %s", version)
	return map[string]interface{}{
		"Simrad SY50 version": version,
	}, nil
}


func (s *sy50updaterSy50Updater) Close(context.Context) error {
	// Put close code here
	s.cancelFunc()
	return nil
}


func (s *sy50updaterSy50Updater) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	var strCaption string
	var strText string
	// Find the Simrad SY50 version in the Windows Registry
	programName := "Simrad SY50"
	version, err := getWindowsProgramVersion(programName)

	s.logger.Infof("Comparing Simrad SY50 installed version %s with desired version %s", version, s.cfg.versionTarget)
	if err != nil {
		// Simrad SY50 not installed or not found in Windows Registry
		strCaption = "Install Simrad SY50"
		strText = "Would you like to download and install the latest Simrad SY50 update?"
	} else {
		strCaption = "Simrad SY50 Update Available"
		strText = fmt.Sprintf("Simrad SY50 %s is installed but an %s update is available.\n\nWould you like to download and install the latest Simrad SY50 update?", version, s.cfg.versionTarget)
	}

	notification := toast.Notification{
		AppID:   "Simrad SY50 Installer",
		Title:   strCaption,
		Message: strText,
		Actions: []toast.Action{
			{Type: "protocol", Label: "Yes",     Arguments: "https://app.viam.com/"}, // Protocol handler will invoke the default browser
			{Type: "protocol", Label: "No",      Arguments: ""},
			{Type: "protocol", Label: "Dismiss", Arguments: ""},
		},
		Duration: "long",
	}

	s.logger.Info("Calling Windows Toast notification...")
	errToast := notification.Push()
	if errToast != nil {
		s.logger.Errorf("error while pushing toast: %v", errToast)
	}
	time.Sleep(10 * time.Second)

	strText = fmt.Sprintf("error while pushing toast: %v", errToast)

	return map[string]interface{}{
		"Simrad SY50 PopUp ": strText,
	}, nil
}


func (s *sy50updaterSy50Updater) DoCommandOld(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messageboxw
	var user32DLL = syscall.NewLazyDLL("user32.dll")
	var procMessageBox = user32DLL.NewProc("MessageBoxW") // Return value: Type int
	const (
		MB_OK          = 0x00000000
		MB_OKCANCEL    = 0x00000001
		MB_YESNO       = 0x00000004
		MB_YESNOCANCEL = 0x00000003

		MB_APPLMODAL   = 0x00000000
		MB_SYSTEMMODAL = 0x00001000
		MB_TASKMODAL   = 0x00002000

		MB_ICONSTOP        = 0x00000010
		MB_ICONQUESTION    = 0x00000020
		MB_ICONWARNING     = 0x00000030
		MB_ICONINFORMATION = 0x00000040
	)

	var strCaption string
	var strText string
	// Find the Simrad SY50 version in the Windows Registry
	programName := "Simrad SY50"
	version, err := getWindowsProgramVersion(programName)
	if err != nil {
		// Not installed or not found
		strCaption = "Install Simrad SY50"
		strText = "Would you like to download and install the latest Simrad SY50 update?"
	} else {
		strCaption = "Simrad SY50 Update Available"
		strText = fmt.Sprintf("Simrad SY50 %s is installed but an update is available.\n\nWould you like to download and install the latest Simrad SY50 update?", version)
	}

	// Debug
	return map[string]interface{}{
		"Simrad SY50 PopUp ": strText,
	}, nil

	lpCaption, _ := syscall.UTF16PtrFromString(strCaption) // LPCWSTR
	lpText, _ := syscall.UTF16PtrFromString(strText)       // LPCWSTR

	// https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messageboxw#return-value
	const (
		IDCANCEL = 2
		IDYES    = 6
		IDNO     = 7
	)

	clickBtnValue, _, _ := syscall.SyscallN(procMessageBox.Addr(),
		0,
		uintptr(unsafe.Pointer(lpText)),
		uintptr(unsafe.Pointer(lpCaption)),
		MB_YESNOCANCEL|
			MB_ICONQUESTION| // You can also choose an icon you like.
			MB_SYSTEMMODAL, // Let the window TOPMOST.
	)

	var retstr string
	if clickBtnValue == IDYES {
		retstr = fmt.Sprintf("Simrad SY50 %s is installed. The user clicked Yes", version)
	} else if clickBtnValue == IDNO {
		retstr = fmt.Sprintf("Simrad SY50 %s is installed. The user clicked No", version)
	} else if clickBtnValue == IDCANCEL {
		retstr = fmt.Sprintf("Simrad SY50 %s is installed. The user clicked Cancel", version)
	}

	return map[string]interface{}{
		"Simrad SY50 PopUp ": retstr,
	}, nil
}


func getWindowsProgramVersion(programName string) (string, error) {
	var subkey string

	// Attempt to find the program's uninstall information in the registry.
	if programName != "" {
		subkey = `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`

		k, err := registry.OpenKey(registry.LOCAL_MACHINE, subkey, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			return "", err
		}
		defer k.Close()

		subkeys, err := k.ReadSubKeyNames(-1)
		if err != nil {
			return "", err
		}

		for _, name := range subkeys {
			appKeyPath := filepath.Join(subkey, name)
			appKey, err := registry.OpenKey(registry.LOCAL_MACHINE, appKeyPath, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			defer appKey.Close()

			displayName, _, err := appKey.GetStringValue("DisplayName")
			if err != nil {
				continue
			}

			if strings.Contains(displayName, programName) {
				version, _, err := appKey.GetStringValue("DisplayVersion")
				if err != nil {
					return "", err
				}
				return version, nil
			}
		}
	}
	return "", fmt.Errorf("program '%s' not found or version information unavailable", programName)
}
