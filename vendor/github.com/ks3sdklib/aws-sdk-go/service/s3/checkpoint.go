package s3

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/rest"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultTaskNum int64 = 3

	MaxPartNum int64 = 10000

	MaxPartSize int64 = 5 * 1024 * 1024 * 1024

	MinPartSize int64 = 100 * 1024

	DefaultPartSize int64 = 5 * 1024 * 1024

	FilePermMode = os.FileMode(0664)

	DirPermMode = os.FileMode(0755)

	CheckpointFileSuffixUploader = ".ucp"

	CheckpointFileSuffixDownloader = ".dcp"

	CheckpointFileSuffixCopier = ".ccp"

	TempFileSuffix = ".temp"

	CheckpointMagic = "B62CAE41-F268-4EC5-839D-FBE475E3FA02"
)

// ------------------------------------ UploadCheckpoint ------------------------------------

type UploadCheckpoint struct {
	Magic                  string
	MD5                    string
	CpFilePath             string           // checkpoint file full path
	UploadFilePath         string           // Local file path
	UploadFileSize         int64            // Local file size
	UploadFileLastModified string           // Local file last modified time
	BucketName             string           // Bucket name
	ObjectKey              string           // Object key
	PartSize               int64            // Part size
	UploadId               string           // Upload ID
	PartETagList           []*CompletedPart // Completed parts
}

func newUploadCheckpoint(u *Uploader) (*UploadCheckpoint, error) {
	request := u.uploadFileRequest
	fileSize := aws.ToLong(request.FileSize)
	partSize := u.getPartSize(fileSize, aws.ToLong(request.PartSize))
	cp := &UploadCheckpoint{
		Magic:          CheckpointMagic,
		UploadFileSize: fileSize,
		BucketName:     aws.ToString(request.Bucket),
		ObjectKey:      aws.ToString(request.Key),
		PartSize:       partSize,
		PartETagList:   make([]*CompletedPart, 0),
	}

	filePath := aws.ToString(request.UploadFile)
	if filePath != "" {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}
		cp.UploadFilePath = filePath
		cp.UploadFileLastModified = fileInfo.ModTime().String()
	} else {
		if request.ObjectMeta != nil {
			cp.UploadFileLastModified = aws.ToString(request.ObjectMeta[HTTPHeaderLastModified])
		}
	}

	return cp, nil
}

func generateUploadCpFilePath(request *UploadFileInput) (string, error) {
	name := fmt.Sprintf("%s/%s", *request.Bucket, *request.Key)
	md5Hash := md5.New()
	md5Hash.Write([]byte("ks3://" + rest.EscapePath(name, false)))
	destHash := hex.EncodeToString(md5Hash.Sum(nil))

	filePath := aws.ToString(request.UploadFile)
	absPath, _ := filepath.Abs(filePath)
	md5Hash.Reset()
	md5Hash.Write([]byte(absPath))
	srcHash := hex.EncodeToString(md5Hash.Sum(nil))

	var dir string
	baseDir := aws.ToString(request.CheckpointDir)
	if baseDir == "" {
		dir = os.TempDir()
	} else {
		dir = filepath.Dir(baseDir)
	}

	cpFilePath := filepath.Join(dir, fmt.Sprintf("%v-%v%v", srcHash, destHash, CheckpointFileSuffixUploader))

	return cpFilePath, nil
}

// load checkpoint from local file
func (cp *UploadCheckpoint) load() error {
	if cp.CpFilePath == "" {
		return nil
	}

	if !FileExists(cp.CpFilePath) {
		return nil
	}

	// 读取断点文件
	contents, err := os.ReadFile(cp.CpFilePath)
	if err != nil {
		return err
	}

	ucp := UploadCheckpoint{}
	if err = json.Unmarshal(contents, &ucp); err != nil {
		return err
	}

	// 判断断点文件是否有效
	if !cp.isValid(ucp) {
		err := cp.remove()
		if err != nil {
			return err
		}
	}

	// 读取断点文件成功，将断点文件中的信息赋值给当前对象
	cp.UploadId = ucp.UploadId
	cp.PartETagList = ucp.PartETagList

	return nil
}

func (cp *UploadCheckpoint) isValid(ucp UploadCheckpoint) bool {
	md5sum := ucp.checksum()
	if CheckpointMagic != ucp.Magic || md5sum != ucp.MD5 {
		return false
	}

	if cp.BucketName != ucp.BucketName ||
		cp.ObjectKey != ucp.ObjectKey ||
		cp.PartSize != ucp.PartSize ||
		cp.UploadFilePath != ucp.UploadFilePath ||
		cp.UploadFileSize != ucp.UploadFileSize ||
		cp.UploadFileLastModified != ucp.UploadFileLastModified {
		return false
	}

	if len(ucp.UploadId) == 0 {
		return false
	}

	return true
}

func (cp *UploadCheckpoint) dump() error {
	if cp.CpFilePath == "" {
		return nil
	}

	dir := filepath.Dir(cp.CpFilePath)
	if !DirExists(dir) {
		err := os.MkdirAll(dir, DirPermMode)
		if err != nil {
			return err
		}
	}

	cp.MD5 = cp.checksum()
	str, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	return os.WriteFile(cp.CpFilePath, str, FilePermMode)
}

func (cp *UploadCheckpoint) checksum() string {
	str := cp.MD5
	cp.MD5 = ""
	json, _ := json.Marshal(cp)
	sum := md5.Sum(json)
	md5sum := hex.EncodeToString(sum[:])
	cp.MD5 = str
	return md5sum
}

func (cp *UploadCheckpoint) remove() error {
	if cp.CpFilePath == "" {
		return nil
	}

	return os.Remove(cp.CpFilePath)
}

// ------------------------------------ DownloadCheckpoint ------------------------------------

type DownloadCheckpoint struct {
	Magic              string
	MD5                string
	CpFilePath         string           // checkpoint file full path
	DownloadFilePath   string           // Local file path
	BucketName         string           // Bucket name
	ObjectKey          string           // Object key
	ObjectSize         int64            // Object size
	ObjectLastModified string           // Object last modified
	PartSize           int64            // Part size
	PartETagList       []*CompletedPart // Completed parts
}

func newDownloadCheckpoint(d *Downloader) (*DownloadCheckpoint, error) {
	request := d.downloadFileRequest
	meta := d.downloadFileMeta
	objectSize, _ := strconv.ParseInt(aws.ToString(meta[HTTPHeaderContentLength]), 10, 64)
	lastModified := aws.ToString(meta[HTTPHeaderLastModified])
	cp := &DownloadCheckpoint{
		Magic:              CheckpointMagic,
		BucketName:         aws.ToString(request.Bucket),
		ObjectKey:          aws.ToString(request.Key),
		DownloadFilePath:   aws.ToString(request.DownloadFile),
		ObjectSize:         objectSize,
		ObjectLastModified: lastModified,
		PartSize:           aws.ToLong(request.PartSize),
		PartETagList:       make([]*CompletedPart, 0),
	}

	return cp, nil
}

func generateDownloadCpFilePath(request *DownloadFileInput) (string, error) {
	name := fmt.Sprintf("%v/%v", *request.Bucket, *request.Key)
	md5Hash := md5.New()
	md5Hash.Write([]byte("ks3://" + rest.EscapePath(name, false)))
	destHash := hex.EncodeToString(md5Hash.Sum(nil))

	filePath := aws.ToString(request.DownloadFile)
	absPath, _ := filepath.Abs(filePath)
	md5Hash.Reset()
	md5Hash.Write([]byte(absPath))
	srcHash := hex.EncodeToString(md5Hash.Sum(nil))

	var dir string
	baseDir := aws.ToString(request.CheckpointDir)
	if baseDir == "" {
		dir = os.TempDir()
	} else {
		dir = filepath.Dir(baseDir)
	}

	cpFilePath := filepath.Join(dir, fmt.Sprintf("%v-%v%v", srcHash, destHash, CheckpointFileSuffixDownloader))

	return cpFilePath, nil
}

// load checkpoint from local file
func (cp *DownloadCheckpoint) load() error {
	if cp.CpFilePath == "" {
		return nil
	}

	if !FileExists(cp.CpFilePath) {
		return nil
	}

	// 读取断点文件
	contents, err := os.ReadFile(cp.CpFilePath)
	if err != nil {
		return err
	}

	dcp := DownloadCheckpoint{}
	if err = json.Unmarshal(contents, &dcp); err != nil {
		return err
	}

	// 判断断点文件是否有效
	if !cp.isValid(dcp) {
		cp.remove()
		return nil
	}

	// 读取断点文件成功，将断点文件中的信息赋值给当前对象
	cp.PartETagList = dcp.PartETagList

	return nil
}

func (cp *DownloadCheckpoint) isValid(dcp DownloadCheckpoint) bool {
	md5sum := dcp.checksum()
	if CheckpointMagic != dcp.Magic || md5sum != dcp.MD5 {
		return false
	}

	if cp.BucketName != dcp.BucketName ||
		cp.ObjectKey != dcp.ObjectKey ||
		cp.PartSize != dcp.PartSize ||
		cp.DownloadFilePath != dcp.DownloadFilePath ||
		cp.ObjectSize != dcp.ObjectSize ||
		cp.ObjectLastModified != dcp.ObjectLastModified {
		return false
	}

	return true
}

func (cp *DownloadCheckpoint) dump() error {
	if cp.CpFilePath == "" {
		return nil
	}

	dir := filepath.Dir(cp.CpFilePath)
	if !DirExists(dir) {
		err := os.MkdirAll(dir, DirPermMode)
		if err != nil {
			return err
		}
	}

	cp.MD5 = cp.checksum()
	str, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	return os.WriteFile(cp.CpFilePath, str, FilePermMode)
}

func (cp *DownloadCheckpoint) checksum() string {
	str := cp.MD5
	cp.MD5 = ""
	json, _ := json.Marshal(cp)
	sum := md5.Sum(json)
	md5sum := hex.EncodeToString(sum[:])
	cp.MD5 = str
	return md5sum
}

func (cp *DownloadCheckpoint) remove() error {
	if cp.CpFilePath == "" {
		return nil
	}

	return os.Remove(cp.CpFilePath)
}

// ------------------------------------ CopyCheckpoint ------------------------------------

type CopyCheckpoint struct {
	Magic                 string
	MD5                   string
	CpFilePath            string           // checkpoint file full path
	BucketName            string           // Bucket name
	ObjectKey             string           // Object key
	SrcBucketName         string           // Source bucket name
	SrcObjectKey          string           // Source object key
	SrcObjectSize         int64            // Source object size
	SrcObjectLastModified string           // Source object last modified time
	PartSize              int64            // Part size
	UploadId              string           // Upload ID
	PartETagList          []*CompletedPart // Completed parts
}

func newCopyCheckpoint(c *Copier) (*CopyCheckpoint, error) {
	request := c.copyFileRequest
	meta := c.copyObjectMeta

	objectSize, _ := strconv.ParseInt(aws.ToString(meta[HTTPHeaderContentLength]), 10, 64)
	lastModified := aws.ToString(meta[HTTPHeaderLastModified])
	partSize := c.getPartSize(objectSize, aws.ToLong(request.PartSize))
	cp := &CopyCheckpoint{
		Magic:                 CheckpointMagic,
		BucketName:            aws.ToString(request.Bucket),
		ObjectKey:             aws.ToString(request.Key),
		SrcBucketName:         aws.ToString(request.SourceBucket),
		SrcObjectKey:          aws.ToString(request.SourceKey),
		SrcObjectSize:         objectSize,
		SrcObjectLastModified: lastModified,
		PartSize:              partSize,
		PartETagList:          make([]*CompletedPart, 0),
	}

	return cp, nil
}

func generateCopyCpFilePath(request *CopyFileInput) (string, error) {
	dstName := fmt.Sprintf("%s/%s", *request.Bucket, *request.Key)
	md5Hash := md5.New()
	md5Hash.Write([]byte("ks3://" + rest.EscapePath(dstName, false)))
	destHash := hex.EncodeToString(md5Hash.Sum(nil))

	srcName := fmt.Sprintf("%s/%s", *request.SourceBucket, *request.SourceKey)
	md5Hash.Reset()
	md5Hash.Write([]byte(srcName))
	srcHash := hex.EncodeToString(md5Hash.Sum(nil))

	var dir string
	baseDir := aws.ToString(request.CheckpointDir)
	if baseDir == "" {
		dir = os.TempDir()
	} else {
		dir = filepath.Dir(baseDir)
	}

	cpFilePath := filepath.Join(dir, fmt.Sprintf("%v-%v%v", srcHash, destHash, CheckpointFileSuffixCopier))

	return cpFilePath, nil
}

// load checkpoint from local file
func (cp *CopyCheckpoint) load() error {
	if cp.CpFilePath == "" {
		return nil
	}

	if !FileExists(cp.CpFilePath) {
		return nil
	}

	// 读取断点文件
	contents, err := os.ReadFile(cp.CpFilePath)
	if err != nil {
		return err
	}

	ccp := CopyCheckpoint{}
	if err = json.Unmarshal(contents, &ccp); err != nil {
		return err
	}

	// 判断断点文件是否有效
	if !cp.isValid(ccp) {
		err := cp.remove()
		if err != nil {
			return err
		}
	}

	// 读取断点文件成功，将断点文件中的信息赋值给当前对象
	cp.UploadId = ccp.UploadId
	cp.PartETagList = ccp.PartETagList

	return nil
}

func (cp *CopyCheckpoint) isValid(ccp CopyCheckpoint) bool {
	md5sum := ccp.checksum()
	if CheckpointMagic != ccp.Magic || md5sum != ccp.MD5 {
		return false
	}

	if cp.BucketName != ccp.BucketName ||
		cp.ObjectKey != ccp.ObjectKey ||
		cp.SrcBucketName != ccp.SrcBucketName ||
		cp.SrcObjectKey != ccp.SrcObjectKey ||
		cp.SrcObjectSize != ccp.SrcObjectSize ||
		cp.SrcObjectLastModified != ccp.SrcObjectLastModified ||
		cp.PartSize != ccp.PartSize {
		return false
	}

	if len(ccp.UploadId) == 0 {
		return false
	}

	return true
}

func (cp *CopyCheckpoint) dump() error {
	if cp.CpFilePath == "" {
		return nil
	}

	dir := filepath.Dir(cp.CpFilePath)
	if !DirExists(dir) {
		err := os.MkdirAll(dir, DirPermMode)
		if err != nil {
			return err
		}
	}

	cp.MD5 = cp.checksum()
	str, err := json.Marshal(cp)
	if err != nil {
		return err
	}

	return os.WriteFile(cp.CpFilePath, str, FilePermMode)
}

func (cp *CopyCheckpoint) checksum() string {
	str := cp.MD5
	cp.MD5 = ""
	json, _ := json.Marshal(cp)
	sum := md5.Sum(json)
	md5sum := hex.EncodeToString(sum[:])
	cp.MD5 = str
	return md5sum
}

func (cp *CopyCheckpoint) remove() error {
	if cp.CpFilePath == "" {
		return nil
	}

	return os.Remove(cp.CpFilePath)
}
