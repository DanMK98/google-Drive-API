package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/drive/v3"
)

// Use Service account
func ServiceAccount(secretFile string) *http.Client {
	b, err := ioutil.ReadFile(secretFile)
	if err != nil {
		log.Fatal("error while reading the credential file", err)
	}
	var s = struct {
		Email      string `json:"client_email"`
		PrivateKey string `json:"private_key"`
	}{}
	if err := json.Unmarshal(b, &s); err != nil {
		log.Fatalf("invalid credential JSON: %v", err)
	}
	config := &jwt.Config{
		Email:      s.Email,
		PrivateKey: []byte(s.PrivateKey),
		Scopes: []string{
			drive.DriveScope,
		},
		TokenURL: google.JWTTokenURL,
	}
	client := config.Client(context.Background())
	return client
}

func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).SupportsAllDrives(true).Do()

	if err != nil {
		log.Println("Could not create file: " + err.Error())
		return nil, err
	}

	return file, nil
}

func checkFolder(ctx context.Context, srv *drive.Service, folderID string) error {
	folder, err := srv.Files.Get(folderID).
		SupportsAllDrives(true).
		Fields("id,name,mimeType,trashed,capabilities(canAddChildren)").
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("cannot access folder %q: %w", folderID, err)
	}

	if folder.Trashed {
		return fmt.Errorf("folder %q is in the trash", folder.Name)
	}

	if folder.MimeType != "application/vnd.google-apps.folder" {
		return fmt.Errorf("ID %q is not a folder", folderID)
	}

	if folder.Capabilities == nil || !folder.Capabilities.CanAddChildren {
		return fmt.Errorf("no permissions to upload into folder %q", folder.Name)
	}

	return nil
}

func main() {
	// Step 1: Open  file
	if len(os.Args) < 2 {
		fmt.Println("Please provide the filename as a command-line argument.")
		return
	} else {
		fmt.Println("File name provided as command-line argument: ", os.Args[1])
	}

	// Step 1: Open file
	filename := os.Args[1]
	f, err := os.Open(filename)

	if err != nil {
		panic(fmt.Sprintf("cannot open file: %v", err))
	} else {
		fmt.Printf("File '%s' opened successfully", filename)
	}

	defer f.Close()

	// Step 2: Get the Google Drive service
	client := ServiceAccount("client_secret.json")

	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}
	// Step 3: Create directory
	// dir, err := createFolder(srv, "New Folder", "root")

	// if err != nil {
	//      panic(fmt.Sprintf("Could not create dir: %v\n", err))
	// }

	//give your folder id here in which you want to upload or create new directory
	folderId := "1QQAZvPo7J76tluo7Q-zIYhkLpfyP3DFn"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := checkFolder(ctx, srv, folderId); err != nil {
		log.Fatal(err)
	}
	// Step 4: create the file and upload
	file, err := createFile(srv, filepath.Base(filename), "application/octet-stream", f, folderId)

	if err != nil {
		panic(fmt.Sprintf("Could not create file: %v\n", err))
	}

	fmt.Printf("File '%s' successfully uploaded", file.Name)
	fmt.Printf("\nFile Id: '%s' ", file.Id)

}
