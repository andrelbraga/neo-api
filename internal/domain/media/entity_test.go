package media

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBrokerPhotoPath(t *testing.T) {
	assert.Equal(t, "", BrokerPhotoPath(""))
	assert.Equal(t, "/public/media/brokers/abc.jpg", BrokerPhotoPath("brokers/abc.jpg"))
}

func TestBrokerKey(t *testing.T) {
	name := "0123456789abcdef0123456789abcdef"
	assert.Equal(t, "brokers/"+name+".jpg", BrokerKey(name+".jpg"))
	assert.Equal(t, "brokers/"+name+".webp", BrokerKey(" "+name+".webp "))
	for _, bad := range []string{"", "../leads/x.jpg", name + ".gif", name + ".jpg/x", "ABCDEF0123456789abcdef0123456789.jpg", "x.jpg"} {
		assert.Empty(t, BrokerKey(bad), bad)
	}
}
