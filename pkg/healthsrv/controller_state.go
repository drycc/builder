package healthsrv

import (
	"net/http"

	"github.com/drycc/builder/pkg/controller"
	drycc "github.com/drycc/controller-sdk-go"
)

// GetClient is an (*net/http).Client compatible interface that provides just the Get cross-section of functionality.
// It can also be implemented for unit tests.
type GetClient interface {
	Get(string) (*http.Response, error)
}

func controllerState(client *drycc.Client, succCh chan<- string, errCh chan<- error, stopCh <-chan struct{}) {
	err := client.Healthcheck()
	if controller.CheckAPICompat(client, err) != nil {
		select {
		case errCh <- err:
		case <-stopCh:
		}
		return
	}
	select {
	case succCh <- "controller is healthy":
	case <-stopCh:
		return
	}
}
