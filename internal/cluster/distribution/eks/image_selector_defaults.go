// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eks

// AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-08eb69ce386c9e5d4",
			"ap-northeast-2": "ami-0b164ef1518193966",
			"ap-northeast-3": "ami-0f86193ab75516a4e",
			"ap-southeast-1": "ami-0c814bff86ffae438",
			"ap-southeast-2": "ami-074d8646b8184300b",
			"ap-south-1":     "ami-01de2b54dee5461ba",
			"ca-central-1":   "ami-07c6ae17e840017d8",
			"eu-central-1":   "ami-0c03b017874e1d7e4",
			"eu-north-1":     "ami-02cc2d248cab6d882",
			"eu-west-1":      "ami-07e16eb3fc2c12328",
			"eu-west-2":      "ami-06178c718a7d9b164",
			"eu-west-3":      "ami-02fe296802edc3bf9",
			"sa-east-1":      "ami-0af82db946859c80f",
			"us-east-1":      "ami-0ad9600b3719f8a53",
			"us-east-2":      "ami-0dce41d8956099d1c",
			"us-west-1":      "ami-08d4cdd9fa56e4cc2",
			"us-west-2":      "ami-0123cdc1d3e4fac7a",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.20
		Constraint: mustConstraint("1.20"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-050c02862539f4984",
			"ap-northeast-2": "ami-0e34fc9c7492756ee",
			"ap-northeast-3": "ami-07acd78236d1e11dc",
			"ap-southeast-1": "ami-0bf1fa15b81394b79",
			"ap-southeast-2": "ami-041460ef88863e87a",
			"ap-south-1":     "ami-0941443c8917f415a",
			"ca-central-1":   "ami-05d993a0162ed26a2",
			"eu-central-1":   "ami-0b273062ba87d6a40",
			"eu-north-1":     "ami-0a89934258901e12e",
			"eu-west-1":      "ami-0e9af5e112a678f08",
			"eu-west-2":      "ami-013244fec0ed69e0c",
			"eu-west-3":      "ami-0878e400a1097c4d4",
			"sa-east-1":      "ami-04cd0cef082181df6",
			"us-east-1":      "ami-01a09362cdb8f50b3",
			"us-east-2":      "ami-0098e60c6f91e0198",
			"us-west-1":      "ami-05a3f9078be6d6b29",
			"us-west-2":      "ami-018dff183a28ac510",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.21
		Constraint: mustConstraint("1.21"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-02c6d763272789974",
			"ap-northeast-2": "ami-0a935a0be13d15a62",
			"ap-northeast-3": "ami-04b6b2ffed6e017cd",
			"ap-southeast-1": "ami-06c6b04b283f6a360",
			"ap-southeast-2": "ami-04844e9ed76402c4c",
			"ap-south-1":     "ami-083fc9a94c76fcf99",
			"ca-central-1":   "ami-08fbcecd888e28020",
			"eu-central-1":   "ami-0cdaae2396feeac04",
			"eu-north-1":     "ami-0eb7a1eaa3452a92e",
			"eu-west-1":      "ami-003f91f482a604b6d",
			"eu-west-2":      "ami-03bfc3ec1fbb98b64",
			"eu-west-3":      "ami-055d6d194079fd39c",
			"sa-east-1":      "ami-044c743d56df85e52",
			"us-east-1":      "ami-05911b9b4df1172c7",
			"us-east-2":      "ami-0b1eb76fbce602b88",
			"us-west-1":      "ami-0b231db44d895f441",
			"us-west-2":      "ami-0927a66ff40101d76",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.22
		Constraint: mustConstraint("1.22"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-0df4e9519930dec44",
			"ap-northeast-2": "ami-006da1df7b3f0314c",
			"ap-northeast-3": "ami-09b7c559361080c12",
			"ap-southeast-1": "ami-091a4816b24a28609",
			"ap-southeast-2": "ami-05ed20b703d2f11a1",
			"ap-south-1":     "ami-0e070d28b9ba80b5f",
			"ca-central-1":   "ami-0adb2ff3a246d92f0",
			"eu-central-1":   "ami-00a3108f45c6dbb6c",
			"eu-north-1":     "ami-03e9fa5d35116a5df",
			"eu-west-1":      "ami-0123eaf3fc3256084",
			"eu-west-2":      "ami-0d184db0f7d78e409",
			"eu-west-3":      "ami-0572799da7d7711e5",
			"sa-east-1":      "ami-0a459177cd5b03219",
			"us-east-1":      "ami-0281494f7c4eb37d1",
			"us-east-2":      "ami-05c1b5a6e6a6fa16e",
			"us-west-1":      "ami-0642b600309077ef5",
			"us-west-2":      "ami-0c8f79982ab160bad",
		},
	},
}

// GPU accelerated AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultAcceleratedImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-083a65dd1e23940d6",
			"ap-northeast-2": "ami-071c054807db2f503",
			"ap-northeast-3": "ami-07e43543f8101065e",
			"ap-southeast-1": "ami-08971297bd9468166",
			"ap-southeast-2": "ami-0ab622a48e9c8f1bf",
			"ap-south-1":     "ami-0a47482c80f025609",
			"ca-central-1":   "ami-0601ebb976725514a",
			"eu-central-1":   "ami-0f6bee88837f9b61d",
			"eu-north-1":     "ami-0f68f9c2858c445b5",
			"eu-west-1":      "ami-0be1bbe01808bb0a4",
			"eu-west-2":      "ami-0b1c67ea57e16a5f2",
			"eu-west-3":      "ami-0e9ccc37532caf7e9",
			"sa-east-1":      "ami-0cf49c14e8c44e9a2",
			"us-east-1":      "ami-0e3321c1d07afe8fa",
			"us-east-2":      "ami-08271bc378986e891",
			"us-west-1":      "ami-0b57966fb6bc0fd23",
			"us-west-2":      "ami-0ca35da729a4ac40a",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.20
		Constraint: mustConstraint("1.20"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-0125f1e98205d987c",
			"ap-northeast-2": "ami-032298e44be552feb",
			"ap-northeast-3": "ami-01241bbc2d5c4d125",
			"ap-southeast-1": "ami-0a16cc6d8966266f1",
			"ap-southeast-2": "ami-069527e61a88bcb5d",
			"ap-south-1":     "ami-0ee26d3bfd6bf38ec",
			"ca-central-1":   "ami-0fd31ad09dd7fd456",
			"eu-central-1":   "ami-07fd1e86fd351f6da",
			"eu-north-1":     "ami-058205a790d275594",
			"eu-west-1":      "ami-0e15713d58d8e5211",
			"eu-west-2":      "ami-0648c6a2d6ae41e9c",
			"eu-west-3":      "ami-0dc3748cb601a17e3",
			"sa-east-1":      "ami-0c10085d70fd4cc1b",
			"us-east-1":      "ami-05ff34e4e99cd267b",
			"us-east-2":      "ami-0e126751a6c757141",
			"us-west-1":      "ami-048f855e1ede0283b",
			"us-west-2":      "ami-05c88d1915d167c2f",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.21
		Constraint: mustConstraint("1.21"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-0a4990c754f9d32c7",
			"ap-northeast-2": "ami-01933704bdfc44321",
			"ap-northeast-3": "ami-0bf1a1c6bb2e510f6",
			"ap-southeast-1": "ami-02b6bf5b3120e1b63",
			"ap-southeast-2": "ami-04ead453a42ba9db2",
			"ap-south-1":     "ami-068fe04611cd93a8d",
			"ca-central-1":   "ami-0dd3e67c0adad0213",
			"eu-central-1":   "ami-0a478b8dec88bc3eb",
			"eu-north-1":     "ami-076525574913be1d0",
			"eu-west-1":      "ami-0ed3ad0ebf44bcf9b",
			"eu-west-2":      "ami-08a1b3c9472c7878c",
			"eu-west-3":      "ami-068c84e8d060e6e22",
			"sa-east-1":      "ami-09f86fe3d726aae42",
			"us-east-1":      "ami-09cae16b82dde1153",
			"us-east-2":      "ami-063c31f4d70721673",
			"us-west-1":      "ami-015de5702e6b02b17",
			"us-west-2":      "ami-0e6099251bce8b29c",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.22
		Constraint: mustConstraint("1.22"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-08b37b86fb4fd0af7",
			"ap-northeast-2": "ami-0e1665ecc92861fd7",
			"ap-northeast-3": "ami-0df38396ff6188d57",
			"ap-southeast-1": "ami-06c4f9ddc24333e21",
			"ap-southeast-2": "ami-0ed9d85ddd847b8c7",
			"ap-south-1":     "ami-07416c807540f19e0",
			"ca-central-1":   "ami-02fe430cb614c963f",
			"eu-central-1":   "ami-0f5f53d4309d7d565",
			"eu-north-1":     "ami-0d61ff815c3c9d6a5",
			"eu-west-1":      "ami-0c9cd2d481133887c",
			"eu-west-2":      "ami-0e317120b8842d186",
			"eu-west-3":      "ami-0ef9794eb51555a8f",
			"sa-east-1":      "ami-0ddd3bca99d897e1a",
			"us-east-1":      "ami-03b85ddc5b923bb9f",
			"us-east-2":      "ami-0c5ab0c5cb252510b",
			"us-west-1":      "ami-004ac117670385f65",
			"us-west-2":      "ami-0e79cbcba993635f2",
		},
	},
}

// ARM architecture AMIs taken form https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
// nolint: gochecknoglobals
var defaultARMImages = ImageSelectors{
	KubernetesVersionImageSelector{ // Kubernetes Version 1.19
		Constraint: mustConstraint("1.19"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-08fff56a9ad60236c",
			"ap-northeast-2": "ami-0ca565eff76235a30",
			"ap-northeast-3": "ami-011549fbeb7aa8fba",
			"ap-southeast-1": "ami-0cb1793755d056e1a",
			"ap-southeast-2": "ami-042a60405a351f554",
			"ap-south-1":     "ami-044a486e18c3b620a",
			"ca-central-1":   "ami-0459d2d0a90ec9801",
			"eu-central-1":   "ami-04d256682d256a07a",
			"eu-north-1":     "ami-040c41e649e8621c9",
			"eu-west-1":      "ami-06079da61b5122a83",
			"eu-west-2":      "ami-0290f1996cdab11f8",
			"eu-west-3":      "ami-0a43a1e41d98d9421",
			"sa-east-1":      "ami-025cac987c3afaac2",
			"us-east-1":      "ami-022d3b138aab73d98",
			"us-east-2":      "ami-0b5adab99b2f4375d",
			"us-west-1":      "ami-07d0dff9fd626e25c",
			"us-west-2":      "ami-0d75141e1d66e526e",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.20
		Constraint: mustConstraint("1.20"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-0d2336687dfca894e",
			"ap-northeast-2": "ami-09986f4223eda7c48",
			"ap-northeast-3": "ami-00e4f962e45e9c17c",
			"ap-southeast-1": "ami-0c690717861dbb0d0",
			"ap-southeast-2": "ami-04d9516d63a4246b5",
			"ap-south-1":     "ami-0098298dcdb4d4f73",
			"ca-central-1":   "ami-0b19e11d1ec38c735",
			"eu-central-1":   "ami-0b23d9c8192758c81",
			"eu-north-1":     "ami-00e05ea0227cef11d",
			"eu-west-1":      "ami-0c39f0b0b72616a26",
			"eu-west-2":      "ami-0456c3b5e01fd3907",
			"eu-west-3":      "ami-0f86f256e9e3e3c0e",
			"sa-east-1":      "ami-004a12ae089b81eb7",
			"us-east-1":      "ami-07872f6b96fa5bccd",
			"us-east-2":      "ami-0f467ecac36fd0e36",
			"us-west-1":      "ami-0c74fdc21383cd0e2",
			"us-west-2":      "ami-0d847ed8a6adf1b45",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.21
		Constraint: mustConstraint("1.21"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-05f84b09bb6c85348",
			"ap-northeast-2": "ami-06b59a38718ad9a78",
			"ap-northeast-3": "ami-0609c606569a668af",
			"ap-southeast-1": "ami-0f241c64bebbb0a44",
			"ap-southeast-2": "ami-09e2180fa3363474c",
			"ap-south-1":     "ami-0c1df4350bc36522d",
			"ca-central-1":   "ami-078950d7db1d9368f",
			"eu-central-1":   "ami-0c7f2c49ac09a6e9a",
			"eu-north-1":     "ami-07f8f8acb3009e55b",
			"eu-west-1":      "ami-0cef76ef3e9890cb7",
			"eu-west-2":      "ami-089a93413ce8e9f74",
			"eu-west-3":      "ami-038dcb2e8df46550a",
			"sa-east-1":      "ami-0033e47aaf815221e",
			"us-east-1":      "ami-027fb45cbec9d71a8",
			"us-east-2":      "ami-01af6c74c327caec2",
			"us-west-1":      "ami-0857aa433ba02b5e6",
			"us-west-2":      "ami-09ebaae9196ae6d53",
		},
	},
	KubernetesVersionImageSelector{ // Kubernetes Version 1.22
		Constraint: mustConstraint("1.22"),
		ImageSelector: RegionMapImageSelector{
			"ap-northeast-1": "ami-09eb62d904b514916",
			"ap-northeast-2": "ami-053988703be2f271e",
			"ap-northeast-3": "ami-00c26bb1646bb6c81",
			"ap-southeast-1": "ami-07837423666de3180",
			"ap-southeast-2": "ami-07f8f0d43565147a7",
			"ap-south-1":     "ami-07d8ef8dbf073a2a0",
			"ca-central-1":   "ami-07f373601dbde68c3",
			"eu-central-1":   "ami-09bca66d746606f7f",
			"eu-north-1":     "ami-0a04f3c375cba7953",
			"eu-west-1":      "ami-0870202aec8e047f3",
			"eu-west-2":      "ami-06ab8799d95434622",
			"eu-west-3":      "ami-0bb5cd92feb0ad801",
			"sa-east-1":      "ami-0f6d8707d9adc389a",
			"us-east-1":      "ami-05179e1815b238699",
			"us-east-2":      "ami-047c76a6b90dc0ccf",
			"us-west-1":      "ami-0f634d1cce0212a78",
			"us-west-2":      "ami-02da13fbaeb77aa58",
		},
	},
}
