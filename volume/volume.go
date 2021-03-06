package volume

import (
	"errors"
	"sync"

	"github.com/libopenstorage/openstorage/api"
)

var (
	ErrExist              = errors.New("Driver already exists")
	ErrDriverNotFound     = errors.New("Driver implementation not found")
	ErrDriverInitializing = errors.New("Driver is initializing")
	ErrEnoEnt             = errors.New("Volume does not exist.")
	ErrEnomem             = errors.New("Out of memory.")
	ErrEinval             = errors.New("Invalid argument")
	ErrVolDetached        = errors.New("Volume is detached")
	ErrVolAttached        = errors.New("Volume is attached")
	ErrVolHasSnaps        = errors.New("Volume has snapshots associated")
	ErrNotSupported       = errors.New("Operation not supported")

	drivers   = make(map[string]func(map[string]string) (VolumeDriver, error))
	instances = make(map[string]VolumeDriver)
	lock      sync.Mutex
)

// VolumeDriver is the main interface to be implemented by any storage driver.
// Every driver must at minimum implement the ProtoDriver sub interface.
type VolumeDriver interface {
	IODriver
	ProtoDriver
	BlockDriver
	Enumerator
}

// IODriver interfaces applicable to object store interfaces.
type IODriver interface {
	// Read sz bytes from specified volume at specified offset.
	// Return number of bytes read and error.
	Read(
		volumeID string,
		buf []byte,
		sz uint64,
		offset int64,
	) (int64, error)

	// Write sz bytes from specified volume at specified offset.
	// Return number of bytes written and error.
	Write(
		volumeID string,
		buf []byte,
		sz uint64,
		offset int64,
	) (int64, error)

	// Flush writes to stable storage.
	// Return error.
	Flush(volumeID string) error
}

// ProtoDriver must be implemented by all volume drivers.  It specifies the
// most basic functionality, such as creating and deleting volumes.
type ProtoDriver interface {
	// String description of this driver.
	String() string

	// Type of this driver
	Type() api.DriverType

	// Create a new Vol for the specific volume spec.
	// It returns a system generated VolumeID that uniquely identifies the volume
	Create(
		locator *api.VolumeLocator,
		Source *api.Source,
		spec *api.VolumeSpec,
	) (string, error)

	// Delete volume.
	// Errors ErrEnoEnt, ErrVolHasSnaps may be returned.
	Delete(volumeID string) error

	// Mount volume at specified path
	// Errors ErrEnoEnt, ErrVolDetached may be returned.
	Mount(volumeID string, mountPath string) error

	// Unmount volume at specified path
	// Errors ErrEnoEnt, ErrVolDetached may be returned.
	Unmount(volumeID string, mountPath string) error

	// Update not all fields of the spec are supported, ErrNotSupported will be thrown for unsupported
	// updates.
	Set(volumeID string, locator *api.VolumeLocator, spec *api.VolumeSpec) error

	// Snapshot create volume snapshot.
	// Errors ErrEnoEnt may be returned
	Snapshot(volumeID string, readonly bool, locator *api.VolumeLocator) (string, error)

	// Stats for specified volume.
	// Errors ErrEnoEnt may be returned
	Stats(volumeID string) (*api.Stats, error)

	// Alerts on this volume.
	// Errors ErrEnoEnt may be returned
	Alerts(volumeID string) (*api.Alerts, error)

	// Status returns a set of key-value pairs which give low
	// level diagnostic status about this driver.
	Status() [][2]string

	// Shutdown and cleanup.
	Shutdown()
}

// Enumerator provides a set of interfaces to get details on a set of volumes.
type Enumerator interface {
	// Inspect specified volumes.
	// Returns slice of volumes that were found.
	Inspect(volumeIDs []string) ([]*api.Volume, error)

	// Enumerate volumes that map to the volumeLocator. Locator fields may be regexp.
	// If locator fields are left blank, this will return all volumes.
	Enumerate(locator *api.VolumeLocator, labels map[string]string) ([]*api.Volume, error)

	// Enumerate snaps for specified volumes
	SnapEnumerate(volID []string, snapLabels map[string]string) ([]*api.Volume, error)
}

// BlockDriver needs to be implemented by block volume drivers.  Filesystem volume
// drivers can ignore this interface and include the builtin DefaultBlockDriver.
type BlockDriver interface {
	// Attach map device to the host.
	// On success the devicePath specifies location where the device is exported
	// Errors ErrEnoEnt, ErrVolAttached may be returned.
	Attach(volumeID string) (string, error)

	// Detach device from the host.
	// Errors ErrEnoEnt, ErrVolDetached may be returned.
	Detach(volumeID string) error
}

func Shutdown() {
	lock.Lock()
	defer lock.Unlock()
	for _, v := range instances {
		v.Shutdown()
	}
}

func Get(name string) (VolumeDriver, error) {
	if v, ok := instances[name]; ok {
		return v, nil
	}
	return nil, ErrDriverNotFound
}

func New(name string, params map[string]string) (VolumeDriver, error) {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := instances[name]; ok {
		return nil, ErrExist
	}
	if initFunc, exists := drivers[name]; exists {
		driver, err := initFunc(params)
		if err != nil {
			return nil, err
		}
		instances[name] = driver
		return driver, err
	}
	return nil, ErrNotSupported
}

func Register(name string, initFunc func(map[string]string) (VolumeDriver, error)) error {
	lock.Lock()
	defer lock.Unlock()
	if _, exists := drivers[name]; exists {
		return ErrExist
	}
	drivers[name] = initFunc
	return nil
}
