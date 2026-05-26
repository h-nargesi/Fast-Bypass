package mikrotik

import "fmt"

func (f *FakeClient) GenerateCertificate(scriptName, title, passphrase string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.certs[title] = passphrase
	f.files[templateOvpnName(title)] = []byte(fmt.Sprintf("client\ndev tun\n# cert cl-%s\n", title))
	return nil
}

func (f *FakeClient) ReadFileContents(name string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.files[name]
	if !ok {
		return nil, ErrNotFound
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

func (f *FakeClient) WriteFileContents(name string, body []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[name] = append([]byte(nil), body...)
	return nil
}

func (f *FakeClient) FileExists(name string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.files[name]
	return ok, nil
}
