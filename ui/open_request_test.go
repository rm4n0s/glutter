package ui

import (
	"reflect"
	"testing"
)

func TestOpenRequestIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		r    OpenRequest
		want bool
	}{
		{"nothing", OpenRequest{}, true},
		{"empty slices", OpenRequest{URLs: []string{}, Files: []string{}}, true},
		{"a URL", OpenRequest{URLs: []string{"magnet:?xt=urn:btih:1"}}, false},
		{"a file", OpenRequest{Files: []string{"/a.torrent"}}, false},
	}
	for _, tt := range tests {
		if got := tt.r.IsEmpty(); got != tt.want {
			t.Errorf("%s: IsEmpty() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestOpenRequestsWaitForTheFirstListener is Theatro's issue (step 56.1): the request that started the app
// comes before the app listens, and is given to the first listener; later ones go to every listener.
func TestOpenRequestsWaitForTheFirstListener(t *testing.T) {
	link := OpenRequest{URLs: []string{"magnet:?xt=urn:btih:1"}}
	file := OpenRequest{Files: []string{"/films/a.torrent"}}
	both := OpenRequest{URLs: []string{"magnet:?xt=urn:btih:2"}, Files: []string{"/films/b.torrent"}}

	d := NewPlatformDispatcher(nil)
	d.DispatchOpenRequest(link)
	d.DispatchOpenRequest(OpenRequest{}) // an empty request is dropped
	d.DispatchOpenRequest(file)

	var first, second []OpenRequest
	removeFirst := d.AddOpenRequestListener(func(r OpenRequest) { first = append(first, r) })
	if want := []OpenRequest{link, file}; !reflect.DeepEqual(first, want) {
		t.Fatalf("the first listener got %+v when it was added, want the requests that waited: %+v", first, want)
	}
	removeSecond := d.AddOpenRequestListener(func(r OpenRequest) { second = append(second, r) })
	if len(second) != 0 {
		t.Errorf("the second listener got %+v, want nothing: the first took what waited", second)
	}

	d.DispatchOpenRequest(both)
	if want := []OpenRequest{link, file, both}; !reflect.DeepEqual(first, want) {
		t.Errorf("the first listener has %+v, want %+v", first, want)
	}
	if want := []OpenRequest{both}; !reflect.DeepEqual(second, want) {
		t.Errorf("the second listener has %+v, want %+v", second, want)
	}

	removeFirst()
	removeFirst() // twice is harmless
	d.DispatchOpenRequest(link)
	if len(first) != 3 {
		t.Errorf("a removed listener was called: %+v", first)
	}
	if want := []OpenRequest{both, link}; !reflect.DeepEqual(second, want) {
		t.Errorf("the second listener has %+v, want %+v", second, want)
	}

	// With no listener left a request waits again.
	removeSecond()
	d.DispatchOpenRequest(file)
	var third []OpenRequest
	d.AddOpenRequestListener(func(r OpenRequest) { third = append(third, r) })
	if want := []OpenRequest{file}; !reflect.DeepEqual(third, want) {
		t.Errorf("the listener added last got %+v, want %+v", third, want)
	}
}

func TestOpenRequestListenersOwnTheirRequest(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	urls := []string{"magnet:?xt=urn:btih:1"}
	var a, b OpenRequest
	d.AddOpenRequestListener(func(r OpenRequest) { a = r; r.URLs[0] = "changed by a" })
	d.AddOpenRequestListener(func(r OpenRequest) { b = r })
	d.DispatchOpenRequest(OpenRequest{URLs: urls})
	if urls[0] != "magnet:?xt=urn:btih:1" || b.URLs[0] != "magnet:?xt=urn:btih:1" {
		t.Errorf("a listener changed the request of another: the sender has %q, the second listener %q", urls[0], b.URLs[0])
	}
	if a.URLs[0] != "changed by a" {
		t.Errorf("the first listener has %q", a.URLs[0])
	}
}

func TestAnOpenRequestListenerMayRemoveAnother(t *testing.T) {
	d := NewPlatformDispatcher(nil)
	var removeB func()
	calls := ""
	d.AddOpenRequestListener(func(OpenRequest) { calls += "a"; removeB() })
	removeB = d.AddOpenRequestListener(func(OpenRequest) { calls += "b" })
	d.DispatchOpenRequest(OpenRequest{Files: []string{"/x"}})
	if calls != "a" {
		t.Errorf("calls = %q, want only the first listener, which removed the second", calls)
	}
}
