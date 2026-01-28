package udf

import (
	"io"
)

const SECTOR_SIZE = 2048

type Udf struct {
	r        io.ReaderAt
	isInited bool
	pvd      *PrimaryVolumeDescriptor
	pd       *PartitionDescriptor
	lvd      *LogicalVolumeDescriptor
	fsd      *FileSetDescriptor
	root_fe  *FileEntry
}

func (udf *Udf) PartitionStart() uint64 {
	if udf.pd == nil {
		panic(udf)
	} else {
		return uint64(udf.pd.PartitionStartingLocation)
	}
}

func (udf *Udf) GetReader() io.ReaderAt {
	return udf.r
}

func (udf *Udf) ReadSectors(sectorNumber uint64, sectorsCount uint64) []byte {
	buf := make([]byte, SECTOR_SIZE*sectorsCount)
	readed, err := udf.r.ReadAt(buf[:], int64(SECTOR_SIZE*sectorNumber))
	if err != nil {
		panic(err)
	}
	if readed != int(SECTOR_SIZE*sectorsCount) {
		panic(readed)
	}
	return buf[:]
}

func (udf *Udf) ReadSector(sectorNumber uint64) []byte {
	return udf.ReadSectors(sectorNumber, 1)
}

func (udf *Udf) init() {
	if udf.isInited {
		return
	}

	anchorDesc := NewAnchorVolumeDescriptorPointer(udf.ReadSector(256))
	if anchorDesc.Descriptor.TagIdentifier != DESCRIPTOR_ANCHOR_VOLUME_POINTER {
		panic(anchorDesc.Descriptor.TagIdentifier)
	}

	for sector := uint64(anchorDesc.MainVolumeDescriptorSeq.Location); ; sector++ {
		desc := NewDescriptor(udf.ReadSector(sector))
		if desc.TagIdentifier == DESCRIPTOR_TERMINATING {
			break
		}
		switch desc.TagIdentifier {
		case DESCRIPTOR_PRIMARY_VOLUME:
			udf.pvd = desc.PrimaryVolumeDescriptor()
		case DESCRIPTOR_PARTITION:
			udf.pd = desc.PartitionDescriptor()
		case DESCRIPTOR_LOGICAL_VOLUME:
			udf.lvd = desc.LogicalVolumeDescriptor()
		}
	}

	partitionStart := udf.PartitionStart()

	udf.fsd = NewFileSetDescriptor(udf.ReadSector(partitionStart + udf.lvd.LogicalVolumeContentsUse.Location))
	udf.root_fe = NewFileEntry(udf.ReadSector(partitionStart + udf.fsd.RootDirectoryICB.Location))

	udf.isInited = true
}

func (udf *Udf) ReadDir(fe *FileEntry) []File {
	udf.init()

	if fe == nil {
		fe = udf.root_fe
	}

	ps := udf.PartitionStart()

	adPos := fe.AllocationDescriptors[0]
	fdLen := uint64(adPos.Length)

	fdBuf := udf.ReadSectors(ps+uint64(adPos.Location), (fdLen+SECTOR_SIZE-1)/SECTOR_SIZE)
	fdOff := uint64(0)

	result := make([]File, 0)

	for uint32(fdOff) < adPos.Length {
		fid := NewFileIdentifierDescriptor(fdBuf[fdOff:])
		if fid.FileIdentifier != "" {
			result = append(result, File{
				Udf: udf,
				Fid: fid,
			})
		}
		fdOff += fid.Len()
	}

	return result
}

func NewUdfFromReader(r io.ReaderAt) *Udf {
	udf := &Udf{
		r:        r,
		isInited: false,
	}

	return udf
}
