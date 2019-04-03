package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type schedtagModelHelper struct {
	managers []modules.JointResourceManager
}

func newSchedtagModelHelper(mans ...modules.JointResourceManager) *schedtagModelHelper {
	return &schedtagModelHelper{managers: mans}
}

func (h *schedtagModelHelper) register() {
	for _, man := range h.managers {
		h.list(man.Slave, man.Slave.GetKeyword())
		h.add(man, man.Slave.GetKeyword())
		h.remove(man, man.Slave.GetKeyword())
		h.setTags(man, man.Slave.GetKeyword())
	}
}

func (h *schedtagModelHelper) list(slave modules.Manager, kw string) {
	R(
		&options.SchedtagModelListOptions{},
		fmt.Sprintf("schedtag-%s-list", kw),
		fmt.Sprintf("List all scheduler tag and %s pairs", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelListOptions) error {
			mod, err := modules.GetJointModule2(s, &modules.Schedtags, slave)
			if err != nil {
				return err
			}
			params, err := args.Params()
			if err != nil {
				return err
			}
			var result *modules.ListResult
			if len(args.Schedtag) > 0 {
				result, err = mod.ListDescendent(s, args.Schedtag, params)
			} else {
				result, err = mod.List(s, params)
			}
			if err != nil {
				return err
			}
			printList(result, mod.GetColumns(s))
			return nil
		},
	)
}

func (h *schedtagModelHelper) add(man modules.JointResourceManager, kw string) {
	R(
		&options.SchedtagModelPairOptions{},
		fmt.Sprintf("schedtag-%s-add", kw),
		fmt.Sprintf("Add a schedtag to a %s", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelPairOptions) error {
			schedtag, err := man.Attach(s, args.SCHEDTAG, args.OBJECT, nil)
			if err != nil {
				return err
			}
			printObject(schedtag)
			return nil
		})
}

func (h *schedtagModelHelper) remove(man modules.JointResourceManager, kw string) {
	R(
		&options.SchedtagModelPairOptions{},
		fmt.Sprintf("schedtag-%s-remove", kw),
		fmt.Sprintf("Remove a schedtag to a %s", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagModelPairOptions) error {
			schedtag, err := man.Detach(s, args.SCHEDTAG, args.OBJECT, nil)
			if err != nil {
				return err
			}
			printObject(schedtag)
			return nil
		})
}

func (h *schedtagModelHelper) setTags(man modules.JointResourceManager, kw string) {
	R(
		&options.SchedtagSetOptions{},
		fmt.Sprintf("%s-set-schedtag", kw),
		fmt.Sprintf("Set schedtags to %v", kw),
		func(s *mcclient.ClientSession, args *options.SchedtagSetOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := man.Slave.PerformAction(s, args.ID, "set-schedtag", params)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})
}

func init() {
	newSchedtagModelHelper(
		modules.Schedtaghosts,
		modules.Schedtagstorages,
	).register()
}
