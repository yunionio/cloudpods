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
	storageaccount *SStorageAccount
	Name           string
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

func (self *SRegion) GetContainers(storageaccount *SStorageAccount) ([]SContainer, error) {
	containers := []SContainer{}
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
	blobClient := client.GetBlobService()
	params := storage.ListContainersParameters{}
	if result, err := blobClient.ListContainers(params); err != nil {
		return nil, err
	} else if err := jsonutils.Update(&containers, result.Containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (self *SRegion) GetContainerDetail(storageaccount *SStorageAccount, containerName string) (*SContainer, error) {
	container := SContainer{}
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
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

func (self *SRegion) CreateContainer(storageaccount *SStorageAccount, containerName string) (*SContainer, error) {
	container, err := self.GetContainerDetail(storageaccount, containerName)
	if err == nil {
		return container, nil
	}
	if err == cloudprovider.ErrNotFound {
		container := SContainer{}
		accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
		if err != nil {
			return nil, err
		}
		client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
		if err != nil {
			return nil, err
		}
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
	return nil, err
}

func (self *SRegion) GetContainerBlobs(storageaccount *SStorageAccount, containerName string) ([]SBlob, error) {
	blobs := []SBlob{}
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
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

func (self *SRegion) GetContainerBlobDetail(storageaccount *SStorageAccount, containerName, blobName string) (*SBlob, error) {
	blob := SBlob{}
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
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

func (self *SRegion) DeleteContainerBlob(storageaccount *SStorageAccount, containerName, blobName string) error {
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return err
	}
	blobClient := client.GetBlobService()
	containerRef := blobClient.GetContainerReference(containerName)
	blobRef := containerRef.GetBlobReference(blobName)
	option := storage.DeleteBlobOptions{}
	if _, err := blobRef.DeleteIfExists(&option); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) CreateBlobFromSnapshot(stoargeaccount *SStorageAccount, containerName, snapshotId string) (*SBlob, error) {
	blob := SBlob{}
	lastIndex := strings.LastIndex(snapshotId, "/")
	blobName := snapshotId[lastIndex+1 : len(snapshotId)]
	self.DeleteContainerBlob(stoargeaccount, containerName, "blobName")
	accessKey, err := self.GetStorageAccountKey(stoargeaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(stoargeaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
	_, err = self.CreateContainer(stoargeaccount, containerName)
	if err != nil {
		return nil, err
	}

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

func (self *SRegion) CreatePageBlob(storageaccount *SStorageAccount, containerName, localPath string) (*SBlob, error) {
	blob := SBlob{}
	blobName := path.Base(localPath)
	self.DeleteContainerBlob(storageaccount, containerName, blobName)
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return nil, err
	}
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

func (self *SRegion) UploadVHD(storageaccount *SStorageAccount, containerName, localVHDPath string) (string, error) {
	if err := ensureVHDSanity(localVHDPath); err != nil {
		return "", err
	}
	diskStream, err := diskstream.CreateNewDiskStream(localVHDPath)
	if err != nil {
		return "", err
	}
	defer diskStream.Close()
	blobName := path.Base(localVHDPath)
	_, err = self.CreateContainer(storageaccount, containerName)
	if err != nil {
		return "", err
	}

	self.DeleteContainerBlob(storageaccount, containerName, blobName)
	accessKey, err := self.GetStorageAccountKey(storageaccount.ID)
	if err != nil {
		return "", err
	}
	client, err := storage.NewBasicClientOnSovereignCloud(storageaccount.Name, accessKey, self.client.env)
	if err != nil {
		return "", err
	}
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
