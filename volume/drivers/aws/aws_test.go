package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/libopenstorage/openstorage/api"
	"github.com/libopenstorage/openstorage/volume"
	"github.com/libopenstorage/openstorage/volume/drivers/test"
)

func TestAll(t *testing.T) {
	if _, err := credentials.NewEnvCredentials().Get(); err != nil {
		t.Skip("No AWS credentials, skipping AWS driver test: ", err)
	}
	_, err := volume.New(Name, map[string]string{})
	if err != nil {
		t.Logf("Failed to initialize Driver: %v", err)
	}
	d, err := volume.Get(Name)
	if err != nil {
		t.Fatalf("Failed to initialize Volume Driver: %v", err)
	}
	ctx := test.NewContext(d)
	ctx.Filesystem = api.FSType_FS_TYPE_EXT4
	test.RunShort(t, ctx)
}
