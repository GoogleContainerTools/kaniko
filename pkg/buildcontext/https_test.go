package buildcontext

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildWithHttpsTar(t *testing.T) {

	tests := []struct {
		name          string
		serverHandler http.HandlerFunc
	}{
		{
			name: "test http bad status",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte("corrupted message"))
				if err != nil {
					t.Fatalf("Error sending response: %v", err)
				}
			}),
		},
		{
			name: "test http bad data",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("corrupted message"))
				if err != nil {
					t.Fatalf("Error sending response: %v", err)
				}
			}),
		},
	}

	for _, tcase := range tests {
		t.Run(tcase.name, func(t *testing.T) {
			server := httptest.NewServer(tcase.serverHandler)
			defer server.Close()

			context := &HTTPSTar{
				context: server.URL + "/data.tar.gz",
			}

			_, err := context.UnpackTarFromBuildContext()
			if err == nil {
				t.Fatalf("Error expected but not returned: %s", err)
			}
		})
	}
}
