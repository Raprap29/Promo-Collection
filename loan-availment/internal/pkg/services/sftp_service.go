package services

import (
	"fmt"
	"globe/dodrio_loan_availment/configs"
	"globe/dodrio_loan_availment/internal/pkg/logger"

	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpService struct {
}

func NewBlacklistAndWhitelistFromSftpService() *SftpService {
	return &SftpService{}
}

func (s *SftpService) sftpConnect() (*sftp.Client, error) {

    // Define the SSH config
	sshConfig := &ssh.ClientConfig{
		User: configs.SFTP_USER,
		Auth: []ssh.AuthMethod{
			ssh.Password(configs.SFTP_PASSWORD),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the SSH server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", configs.SFTP_HOST, configs.SFTP_PORT), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	// Create a new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		// Close the SSH connection if creating the SFTP client fails
		conn.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

    // Return the SFTP client, leaving the connection open
    return client, nil
}

func (s *SftpService) PullCSVFromSFTP() (string, error) {
    client, err := s.sftpConnect()
    if err != nil {
        return "", err
    }
    defer client.Close()

    remoteDir := "/upload/input"
    files, err := client.ReadDir(remoteDir)
    if err != nil {
        return "", err
    }

    var remoteFilePath string
    for _, file := range files {
        if filepath.Ext(file.Name()) == ".csv" {
            remoteFilePath = filepath.Join(remoteDir, file.Name())
            break
        }
    }

    if remoteFilePath == "" {
        return "", nil // No CSV file found
    }

    localFilePath := filepath.Join("temp", filepath.Base(remoteFilePath))
    localFile, err := os.Create(localFilePath)
    if err != nil {
        return "", err
    }
    defer localFile.Close()

    remoteFile, err := client.Open(remoteFilePath)
    if err != nil {
        return "", err
    }
    defer remoteFile.Close()

    if _, err := remoteFile.WriteTo(localFile); err != nil {
        return "", err
    }

    logger.Info("localFilePath in PullCSVFromSFTP is: %v", localFilePath)

    return localFilePath, nil
}

func (s *SftpService) MoveFileOnSFTP(srcPath, destPath string) error {
    client, err := s.sftpConnect()
    if err != nil {
        return err
    }
    defer client.Close()

    // Ensure the processed folder exists
    destDir := filepath.Dir(destPath)
    if _, err := client.Stat(destDir); os.IsNotExist(err) {
        err = client.MkdirAll(destDir)
        if err != nil {
            return fmt.Errorf("failed to create directory: %v", err)
        }
    }

    // Move the file on the SFTP server
    err = client.Rename(srcPath, destPath)
    if err != nil {
        return fmt.Errorf("failed to move file: %v", err)
    }

    return nil
}

func (s *SftpService) UploadFileToSFTP(localFilePath, remoteFilePath string) error {
    client, err := s.sftpConnect()
    if err != nil {
        return err
    }

    // Ensure the remote directory exists
    remoteDir := filepath.Dir(remoteFilePath)
    if _, err := client.Stat(remoteDir); os.IsNotExist(err) {
        err = client.MkdirAll(remoteDir)
        if err != nil {
            return fmt.Errorf("failed to create directory on SFTP server: %v", err)
        }
    }

    localFile, err := os.Open(localFilePath)
    if err != nil {
        return fmt.Errorf("could not open local file: %v", err)
    }
    defer localFile.Close()

    remoteFile, err := client.Create(remoteFilePath)
    if err != nil {
        return fmt.Errorf("could not create remote file: %v", err)
    }
    defer remoteFile.Close()

    _, err = localFile.Seek(0, 0)
    if err != nil {
        return fmt.Errorf("could not seek local file: %v", err)
    }

    _, err = remoteFile.ReadFrom(localFile)
    if err != nil {
        return fmt.Errorf("could not upload file to SFTP server: %v", err)
    }

    return nil
}

func (s *SftpService) DeleteFileOnSFTP(filePath string) error {
    client, err := s.sftpConnect()
    if err != nil {
        return err
    }
    defer client.Close()

    err = client.Remove(filePath)
    if err != nil {
        return fmt.Errorf("failed to delete file on SFTP server: %v", err)
    }

    return nil
}

func (s *SftpService) DeleteLocalFile(filePath string) error {
    return os.Remove(filePath)
}