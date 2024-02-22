/*
Based on https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/main/provider/fake/fake_server.go



*/

package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"

	"google.golang.org/grpc"
)

type MockCSIProviderServer struct {
	grpcServer *grpc.Server
	listener   net.Listener
	socketPath string
	valuesPath string
	returnErr  error
	errorCode  string
	objects    []*v1alpha1.ObjectVersion
	files      []*v1alpha1.File
}

// NewMocKCSIProviderServer returns a mock csi-provider grpc server
func NewMocKCSIProviderServer(socketPath string, valuesPath string) (*MockCSIProviderServer, error) {
	log.Println("NewMocKCSIProviderServer called")
	server := grpc.NewServer()
	s := &MockCSIProviderServer{
		grpcServer: server,
		socketPath: socketPath,
		valuesPath: valuesPath,
	}
	v1alpha1.RegisterCSIDriverProviderServer(server, s)
	return s, nil
}

// SetReturnError sets expected error
func (m *MockCSIProviderServer) SetReturnError(err error) {
	log.Println("SetReturnError called")
	m.returnErr = err
}

// SetObjects sets expected objects id and version
func (m *MockCSIProviderServer) SetObjects(objects map[string]string) {
	log.Println("SetObjects called")
	var ov []*v1alpha1.ObjectVersion
	for k, v := range objects {
		ov = append(ov, &v1alpha1.ObjectVersion{Id: k, Version: v})
	}
	m.objects = ov
}

// SetFiles sets provider files to return on Mount
func (m *MockCSIProviderServer) SetFiles(files []*v1alpha1.File) {
	log.Println("SetFiles called")
	var ov []*v1alpha1.File
	for _, v := range files {
		ov = append(ov, &v1alpha1.File{
			Path:     v.Path,
			Mode:     v.Mode,
			Contents: v.Contents,
		})
	}
	m.files = ov
}

// SetProviderErrorCode sets provider error code to return
func (m *MockCSIProviderServer) SetProviderErrorCode(errorCode string) {
	log.Println("SetProviderErrorCode called")
	m.errorCode = errorCode
}

func (m *MockCSIProviderServer) Start() error {
	log.Println("Start called")
	var err error
	_, err = os.Stat(m.socketPath)
	if err == nil {
		log.Println("Removing old socket..")
		os.Remove(m.socketPath)
	}
	m.listener, err = net.Listen("unix", m.socketPath)
	if err != nil {
		return err
	}
	go func() {
		if err = m.grpcServer.Serve(m.listener); err != nil {
			return
		}
	}()
	return nil
}

func (m *MockCSIProviderServer) Stop() {
	log.Println("Stop called")
	m.grpcServer.GracefulStop()
}

// Mount implements provider csi-provider method
func (m *MockCSIProviderServer) Mount(ctx context.Context, req *v1alpha1.MountRequest) (*v1alpha1.MountResponse, error) {
	log.Println("Mount called")
	var attrib, secret map[string]string
	var filePermission os.FileMode
	var err error

	log.Println("Attr: ", req.GetAttributes())
	log.Println("CurrentObjectVersion: ", req.CurrentObjectVersion)
	log.Println("Secrets: ", req.Secrets)
	log.Println("Target Path: ", req.TargetPath)
	log.Println("Permissions: ", req.GetPermission())

	if m.returnErr != nil {
		return &v1alpha1.MountResponse{}, m.returnErr
	}
	if err = json.Unmarshal([]byte(req.GetAttributes()), &attrib); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attributes, error: %w", err)
	}
	if err = json.Unmarshal([]byte(req.GetSecrets()), &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets, error: %w", err)
	}
	if err = json.Unmarshal([]byte(req.GetPermission()), &filePermission); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file permission, error: %w", err)
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, fmt.Errorf("missing target path")
	}
	yml, err := ParseYAML(attrib["objects"])
	if err != nil {
		return nil, err
	}
	log.Println("yml", yml)
	log.Println("secrets", secret)

	files := make([]*v1alpha1.File, 0)
	versions := make([]*v1alpha1.ObjectVersion, 0)
	log.Println(yml)
	for _, f := range yml {
		secret, err := m.loadSecret(f["objectName"])
		if err != nil {
			return nil, err
		}

		files = append(files, &v1alpha1.File{
			Path:     f["objectName"],
			Mode:     511,
			Contents: secret,
		})
		versions = append(versions, &v1alpha1.ObjectVersion{
			Id:      f["objectName"],
			Version: "1",
		})
	}

	m.SetFiles(files)

	return &v1alpha1.MountResponse{
		ObjectVersion: versions,
		Files:         files,
	}, nil
}

func (m *MockCSIProviderServer) loadSecret(objectName string) ([]byte, error) {
	return os.ReadFile(filepath.Join(m.valuesPath, objectName))

}

// Version implements provider csi-provider method
func (m *MockCSIProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	log.Println("Version called")
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeName:    "fakeprovider",
		RuntimeVersion: "0.0.10",
	}, nil
}

func ParseYAML(attr string) ([]map[string]string, error) {
	yml := make(map[any][]any, 0)
	err := yaml.Unmarshal([]byte(attr), yml)
	log.Println(err, yml)

	for key, value := range yml {
		log.Printf("key (%T): %v", key, key)
		log.Printf("value (%T): %v", value, value)
	}
	a := yml["array"]
	final := make([]map[string]string, 0, len(a))
	for _, value := range a {
		log.Printf("value (%T): %v", value, value)
		m := make(map[string]string)
		yaml.Unmarshal([]byte(value.(string)), &m)
		log.Println("map: ", m)
		final = append(final, m)
	}

	return final, err
}
