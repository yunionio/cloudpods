package azure

import (
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/common"
	"github.com/Microsoft/azure-vhd-utils/vhdcore/diskstream"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SContainer struct {
	Name string
}

type BlobProperties struct {
	LastModified          time.Time
	ContentMD5            string
	ContentLength         int64
	ContentType           string
	ContentEncoding       string
	CacheControl          string
	ContentLanguage       string
	ContentDisposition    string
	BlobType              string
	SequenceNumber        int64
	CopyID                string
	CopyStatus            string
	CopySource            string
	CopyProgress          string
	CopyCompletionTime    time.Time
	CopyStatusDescription string
	LeaseStatus           string
	LeaseState            string
	LeaseDuration         string
	ServerEncrypted       bool
	IncrementalCopy       bool
}

type SBlob struct {
	Name       string
	Snapshot   time.Time
	Properties BlobProperties
	Metadata   map[string]string
}

func (self *SRegion) GetContainers(accountId string) ([]SContainer, error) {
	containers := []SContainer{}
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		params := storage.ListContainersParameters{}
		if result, err := blobClient.ListContainers(params); err != nil {
			return nil, err
		} else if err := jsonutils.Update(&containers, result.Containers); err != nil {
			return nil, err
		}
		return containers, nil
	}
}

func (self *SRegion) GetContainerDetail(accountId string, containerName string) (*SContainer, error) {
	container := SContainer{}
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		containerClient := blobClient.GetContainerReference(containerName)
		if err := containerClient.GetProperties(); err != nil {
			if strings.Index(err.Error(), "StatusCode=404") > 0 {
				return nil, cloudprovider.ErrNotFound
			}
			return nil, err
		} else if err := jsonutils.Update(&container, containerClient); err != nil {
			return nil, err
		}
		return &container, nil
	}
}

func (self *SRegion) CreateContainer(accountId string, containerName string) (*SContainer, error) {
	if container, err := self.GetContainerDetail(accountId, containerName); err == nil {
		return container, nil
	} else if err == cloudprovider.ErrNotFound {
		container := SContainer{}
		globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
		if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
			return nil, err
		} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
			return nil, err
		} else {
			blobClient := client.GetBlobService()
			containerClient := blobClient.GetContainerReference(containerName)
			option := storage.CreateContainerOptions{
				Access: storage.ContainerAccessTypeContainer,
			}
			if err := containerClient.Create(&option); err != nil {
				return nil, err
			}
			if err := containerClient.GetProperties(); err != nil {
				if strings.Index(err.Error(), "StatusCode=404") > 0 {
					return nil, cloudprovider.ErrNotFound
				}
				return nil, err
			} else if jsonutils.Update(&container, containerClient); err != nil {
				return nil, err
			}
			return &container, nil
		}
	} else {
		return nil, err
	}
}

func (self *SRegion) GetContainerBlobs(accountId, containerName string) ([]SBlob, error) {
	blobs := []SBlob{}
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		containerClient := blobClient.GetContainerReference(containerName)
		params := storage.ListBlobsParameters{}
		if result, err := containerClient.ListBlobs(params); err != nil {
			return nil, err
		} else if err := jsonutils.Update(&blobs, result.Blobs); err != nil {
			return nil, err
		}
		return blobs, nil
	}
}

func (self *SRegion) GetContainerBlobDetail(accountId, containerName, blobName string) (*SBlob, error) {
	blob := SBlob{}
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		containerRef := blobClient.GetContainerReference(containerName)
		blobRef := containerRef.GetBlobReference(blobName)
		option := storage.GetBlobPropertiesOptions{}
		if err := blobRef.GetProperties(&option); err != nil {
			if strings.Index(err.Error(), "StatusCode=404") > 0 {
				return nil, cloudprovider.ErrNotFound
			}
			return nil, err
		} else if err := jsonutils.Update(&blob, blobRef); err != nil {
			return nil, err
		}
		return &blob, nil
	}
}

func (self *SRegion) DeleteContainerBlob(accountId, containerName, blobName string) error {
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return err
	} else {
		blobClient := client.GetBlobService()
		containerRef := blobClient.GetContainerReference(containerName)
		blobRef := containerRef.GetBlobReference(blobName)
		option := storage.DeleteBlobOptions{}
		if _, err := blobRef.DeleteIfExists(&option); err != nil {
			return err
		}
		return nil
	}
}

func (self *SRegion) CreateBlobFromSnapshot(accountId, containerName, snapshotId string) (*SBlob, error) {
	blob := SBlob{}
	_, _, blobName := pareResourceGroupWithName(snapshotId, SNAPSHOT_RESOURCE)
	self.DeleteContainerBlob(accountId, containerName, blobName)
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		containerRef := blobClient.GetContainerReference(containerName)
		blobRef := containerRef.GetBlobReference(blobName)
		if uri, err := self.GrantAccessSnapshot(snapshotId); err != nil {
			return nil, err
		} else if err := blobRef.Copy(uri, &storage.CopyOptions{}); err != nil {
			return nil, err
		}
		if err := blobRef.GetProperties(&storage.GetBlobPropertiesOptions{}); err != nil {
			return nil, err
		} else if err := jsonutils.Update(&blob, blobRef); err != nil {
			return nil, err
		}
		return &blob, nil
	}
}

// func (self *SRegion) DownloadPageBlob(accountId, containerName, blobName, output string) error {
// 	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
// 	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
// 		return err
// 	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
// 		return err
// 	} else {
// 		blobClient := client.GetBlobService()
// 		containerRef := blobClient.GetContainerReference(containerName)
// 		blobRef := containerRef.GetBlobReference(blobName)
// 		if downloadLink, err := blobRef.GetSASURI(storage.BlobSASOptions{
// 			BlobServiceSASPermissions: storage.BlobServiceSASPermissions{
// 				Read: true,
// 			},
// 			SASOptions: storage.SASOptions{
// 				Start:  time.Now(),
// 				Expiry: time.Now().Add(time.Hour * 24),
// 				//IP:       "*",
// 				UseHTTPS: true,
// 			},
// 		}); err != nil {
// 			return err
// 		} else {
// 			log.Errorf("donwload link %s", downloadLink)
// 		}
// 	}
// 	return nil
// }

func (self *SRegion) CreatePageBlob(accountId, containerName, localPath string) (*SBlob, error) {
	blob := SBlob{}
	blobName := path.Base(localPath)
	self.DeleteContainerBlob(accountId, containerName, blobName)
	globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
	if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
		return nil, err
	} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
		return nil, err
	} else {
		blobClient := client.GetBlobService()
		containerRef := blobClient.GetContainerReference(containerName)
		blobRef := containerRef.GetBlobReference(blobName)
		//blobRef.Properties.ContentLength =
		if err := blobRef.PutPageBlob(&storage.PutBlobOptions{}); err != nil {
			return nil, err
		}
		if err := blobRef.GetProperties(&storage.GetBlobPropertiesOptions{}); err != nil {
			return nil, err
		} else if err := jsonutils.Update(&blob, blobRef); err != nil {
			return nil, err
		}
		return &blob, nil
	}
}

func (self *SRegion) UploadVHD(accountId, containerName, localVHDPath string) (string, error) {
	if err := ensureVHDSanity(localVHDPath); err != nil {
		return "", err
	}
	if diskStream, err := diskstream.CreateNewDiskStream(localVHDPath); err != nil {
		return "", err
	} else {
		defer diskStream.Close()
		blobName := path.Base(localVHDPath)
		self.DeleteContainerBlob(accountId, containerName, blobName)
		globalId, _, storageAccount := pareResourceGroupWithName(accountId, STORAGE_RESOURCE)
		if accessKey, err := self.GetStorageAccountKey(globalId); err != nil {
			return "", err
		} else if client, err := storage.NewBasicClientOnSovereignCloud(storageAccount, accessKey, self.client.env); err != nil {
			return "", err
		} else {
			blobClient := client.GetBlobService()
			containerRef := blobClient.GetContainerReference(containerName)
			blobRef := containerRef.GetBlobReference(blobName)
			blobRef.Properties.ContentLength = diskStream.GetSize()
			if err := blobRef.PutPageBlob(&storage.PutBlobOptions{}); err != nil {
				return "", err
			}

			var rangesToSkip []*common.IndexRange
			uploadableRanges, err := LocateUploadableRanges(diskStream, rangesToSkip, DefaultReadBlockSize)
			if err != nil {
				return "", err
			}
			if uploadableRanges, err = DetectEmptyRanges(diskStream, uploadableRanges); err != nil {
				return "", err
			}

			cxt := &DiskUploadContext{
				VhdStream:             diskStream,
				UploadableRanges:      uploadableRanges,
				AlreadyProcessedBytes: common.TotalRangeLength(rangesToSkip),
				BlobServiceClient:     blobClient,
				ContainerName:         containerName,
				BlobName:              blobName,
				Parallelism:           3,
				Resume:                false,
				MD5Hash:               []byte(""), //localMetaData.FileMetaData.MD5Hash,
			}

			if err := Upload(cxt); err != nil {
				return "", err
			}

			return blobRef.GetURL(), nil
		}
	}
}
