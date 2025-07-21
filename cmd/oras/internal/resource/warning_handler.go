package resource

import (
	"github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry/remote"
	"sync"
)

var (
	warningHandler WarningHandler
)

type WarningHandler struct {
	once   sync.Once
	warned sync.Map
	logger logrus.FieldLogger
}

func NewWarningHandler(logger logrus.FieldLogger) *WarningHandler {
	warningHandler.once.Do(func() {
		warningHandler.logger = logger
	})
	return &warningHandler
}

func (wh *WarningHandler) GetHandler(registry string) func(warning remote.Warning) {
	result, _ := wh.warned.LoadOrStore(registry, &sync.Map{})
	warner := result.(*sync.Map)
	return func(warning remote.Warning) {
		if _, loaded := warner.LoadOrStore(warning.WarningValue, true); !loaded {
			wh.logger.WithField("registry", registry).Warn(warning.Text)
		}
	}
}
