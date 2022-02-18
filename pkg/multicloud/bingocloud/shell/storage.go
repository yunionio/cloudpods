/*
 * @Author: your name
 * @Date: 2022-02-18 10:37:46
 * @LastEditTime: 2022-02-18 11:06:48
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\shell\storage.go
 */
package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		MaxRecords int
		NextToken  string
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "list storage", func(cli *bingocloud.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetStorages()
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type StorageIdOptions struct {
		ID string
	}

	shellutils.R(&StorageIdOptions{}, "storage-show", "show storage", func(cli *bingocloud.SRegion, args *StorageIdOptions) error {
		storage, err := cli.GetStorage(args.ID)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

}
