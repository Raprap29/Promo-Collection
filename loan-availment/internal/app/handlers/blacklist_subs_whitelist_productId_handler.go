package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"github.com/gin-gonic/gin"

	"globe/dodrio_loan_availment/internal/pkg/services"
)

type SftpHandler struct {
    sftpService services.SFTPClientInterface
}

func NewSftpHandler(sftpService services.SFTPClientInterface) *SftpHandler {
    return &SftpHandler{sftpService: sftpService}
}


func (h *SftpHandler) BlacklistSubs(c *gin.Context) {

	// Step 1: Pull the CSV from SFTP and save it locally
    localFilePath, err := h.sftpService.PullCSVFromSFTP()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if localFilePath == "" {
        c.JSON(http.StatusNotFound, gin.H{"message": "No CSV file found on SFTP server"})
        return
    }

	// Step 2: Validate and update the CSV file
    err = services.UpdateAndValidateCSVFileForBlacklisting(localFilePath)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Step 3: Process the updated CSV and update the database
    err = services.ProcessCSVAndUpdateDBForBlacklistedSubs(c,localFilePath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Generate a timestamp for file naming
	timestamp := time.Now().Format("20060102150405")
	baseFileName := filepath.Base(localFilePath)
	fileNameWithoutExt := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))

    // Define file names with timestamp
	failedFileName := fileNameWithoutExt + "_" + timestamp + ".csv"
	updatedFileName := fileNameWithoutExt + "_" + timestamp + ".csv"

    // Define paths for the failed and updated CSVs
	srcPath := filepath.Join("/upload/input", baseFileName)
	failedFilePath := filepath.Join("temp", failedFileName)
	failedDestPath := filepath.Join("/upload/failed", failedFileName)
	updatedCSVPath := localFilePath 
	updatedCSVDestPath := filepath.Join("/upload/processed", updatedFileName)

	// Delete the original CSV from the input folder on the SFTP server
    err = h.sftpService.DeleteFileOnSFTP(srcPath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete original file from SFTP server", "details": err.Error()})
        return
    }

	 // Upload the failed.csv file to the failed folder on SFTP
	 if _, err := os.Stat(failedFilePath); err == nil { // Check if failed.csv exists
        err = h.sftpService.UploadFileToSFTP(failedFilePath, failedDestPath)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload failed.csv to SFTP server", "details": err.Error()})
            return
        }
    }

    // Upload the updated CSV file to the processed folder
    err = h.sftpService.UploadFileToSFTP(updatedCSVPath, updatedCSVDestPath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload updated CSV to SFTP server", "details": err.Error()})
        return
    }

    // Step 5: Delete the local file from the temp directory
    err = h.sftpService.DeleteLocalFile(localFilePath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete local file", "details": err.Error()})
        return
    }

    if _, err := os.Stat(failedFilePath); err == nil { // Check if failed.csv exists before deletion
        err = h.sftpService.DeleteLocalFile(failedFilePath)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete failed.csv", "details": err.Error()})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "File processed, moved to processed folder, and deleted locally", "file": localFilePath})
}



func (h *SftpHandler) WhitelistSubsForProductId(c *gin.Context) {

    log.Println("I am inside whitelisted subs")

	// Step 1: Pull the CSV from SFTP and save it locally
    localFilePath, err := h.sftpService.PullCSVFromSFTP()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if localFilePath == "" {
        c.JSON(http.StatusNotFound, gin.H{"message": "No CSV file found on SFTP server"})
        return
    }

	// Step 2: Validate and update the CSV file
    productId, err := services.UpdateAndValidateCSVFileForWhitelisting(c,localFilePath)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Step 3: Process the updated CSV and update the database
    err = services.ProcessCSVAndUpdateDBForWhitelistProductId(c,localFilePath, productId)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

	// Generate a timestamp for file naming
	timestamp := time.Now().Format("20060102150405")
	baseFileName := filepath.Base(localFilePath)
	fileNameWithoutExt := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))

    // Define file names with timestamp
	failedFileName := fileNameWithoutExt + "_" + timestamp + ".csv"
	updatedFileName := fileNameWithoutExt + "_" + timestamp + ".csv"

    // Define paths for the failed and updated CSVs
	srcPath := filepath.Join("/upload/input", baseFileName)
	failedFilePath := filepath.Join("temp", failedFileName)
	failedDestPath := filepath.Join("/upload/failed", failedFileName)
	updatedCSVPath := localFilePath 
	updatedCSVDestPath := filepath.Join("/upload/processed", updatedFileName)

	// Delete the original CSV from the input folder on the SFTP server
    err = h.sftpService.DeleteFileOnSFTP(srcPath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete original file from SFTP server", "details": err.Error()})
        return
    }

	 // Upload the failed csv file to the failed folder on SFTP
	 if _, err := os.Stat(failedFilePath); err == nil { // Check if failed.csv exists
        err = h.sftpService.UploadFileToSFTP(failedFilePath, failedDestPath)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload failed.csv to SFTP server", "details": err.Error()})
            return
        }
    }

    // Upload the updated CSV file to the processed folder
    err = h.sftpService.UploadFileToSFTP(updatedCSVPath, updatedCSVDestPath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload updated CSV to SFTP server", "details": err.Error()})
        return
    }

    // Step 5: Delete the local file from the temp directory
    err = h.sftpService.DeleteLocalFile(localFilePath)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete local file", "details": err.Error()})
        return
    }

    if _, err := os.Stat(failedFilePath); err == nil { // Check if failed csv exists before deletion
        err = h.sftpService.DeleteLocalFile(failedFilePath)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete failed csv", "details": err.Error()})
            return
        }
    }

    c.JSON(http.StatusOK, gin.H{"message": "File processed, moved to processed folder, and deleted locally", "file": localFilePath})
}