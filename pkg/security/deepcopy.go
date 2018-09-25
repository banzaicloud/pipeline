package security

import "k8s.io/apimachinery/pkg/runtime"

func (in *WhiteListItem) DeepCopyInto(out *WhiteListItem) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = WhiteListSpec{
		ReleaseName: in.Spec.ReleaseName,
		Creator:     in.Spec.Creator,
		Reason:      in.Spec.Reason,
	}
}

func (in *WhiteListItem) DeepCopyObject() runtime.Object {
	out := WhiteListItem{}
	in.DeepCopyInto(&out)

	return &out
}

func (in *WhiteList) DeepCopyObject() runtime.Object {
	out := WhiteList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]WhiteListItem, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
