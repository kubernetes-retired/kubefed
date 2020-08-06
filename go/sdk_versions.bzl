# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

OVERRIDE_GO_VERSIONS = {
    "1.15.0-rc.1": {
        "darwin_amd64": (
            "go1.15rc1.darwin-amd64.tar.gz",
            "0572e053ed5fd6e8d6ed24f62832b747d46787288e146e8ba99b574b6e0d67b0",
        ),
        "freebsd_386": (
            "go1.15rc1.freebsd-386.tar.gz",
            "479c98371fd29426378596fbc94f96bdc4ac4a9d2bcb4f1ddbc4c1d4edb09ab5",
        ),
        "freebsd_amd64": (
            "go1.15rc1.freebsd-amd64.tar.gz",
            "9b14badd4b8dc881c9a15c2493565107ec92e78a71a51c7251cc0c377f92c3f9",
        ),
        "linux_386": (
            "go1.15rc1.linux-386.tar.gz",
            "e8b09a03cf057fe68806c0d2954ab8d9ca3002558d8ce60a196b836dacb91f4b",
        ),
        "linux_amd64": (
            "go1.15rc1.linux-amd64.tar.gz",
            "ac092ebb92f88366786063e68a9531d5eccac51371f9becb128f064721731b2e",
        ),
        "linux_arm64": (
            "go1.15rc1.linux-arm64.tar.gz",
            "3baf4336d1bcf1c6707c6e2a402a31cbc87cbd9a63687c97c5149911fe0e5beb",
        ),
        "linux_arm": (
            "go1.15rc1.linux-armv6l.tar.gz",
            "d42df2b62fc7569931fb458952b518e1ee102294efcc4e28c54cce76a7f4cd8f",
        ),
        "linux_ppc64le": (
            "go1.15rc1.linux-ppc64le.tar.gz",
            "a8599883755d188d24a5012f72f99b3237c2f5223bc1f937b6f055456c1468e3",
        ),
        "linux_s390x": (
            "go1.15rc1.linux-s390x.tar.gz",
            "0a16994b1f988db12aa44aa9965ae4d07d067489c321e5f7445eb2be63fe2466",
        ),
        "windows_386": (
            "go1.15rc1.windows-386.zip",
            "2e5f90da04f2ba073501eeb7931b897c9d57c9f8e079ee77620c6b1e4f9a8bdf",
        ),
        "windows_amd64": (
            "go1.15rc1.windows-amd64.zip",
            "cc05edc8620ed280dc4540b28312fdd99019a2a14693b6cc9158a26b43e67df3",
        ),
    },
    "1.15.0-beta.1": {
        "darwin_amd64": (
            "go1.15beta1.darwin-amd64.tar.gz",
            "4ee49feb46169ef942097513b5e783ff0f3f276b1eacfc51083e6e453117bd7e",
        ),
        "freebsd_386": (
            "go1.15beta1.freebsd-386.tar.gz",
            "77bc3aae4abaa73b537435b6a497043929cf95d7dd17c289f6e1b55180285c94",
        ),
        "freebsd_amd64": (
            "go1.15beta1.freebsd-amd64.tar.gz",
            "e13dd8a3e5a04bc1a54b2b70f540fd5e4d77663948c14636e27cf8a8ecfccd7b",
        ),
        "linux_386": (
            "go1.15beta1.linux-386.tar.gz",
            "83d732a3961006e058f44c9672fde93dbea3d1c3d69e8807d135eeaf21fb80c8",
        ),
        "linux_amd64": (
            "go1.15beta1.linux-amd64.tar.gz",
            "11814b7475680a09720f3de32c66bca135289c8d528b2e1132b0ce56b3d9d6d7",
        ),
        "linux_arm64": (
            "go1.15beta1.linux-arm64.tar.gz",
            "2648b7d08fe74d0486ec82b3b539d15f3dd63bb34d79e7e57bebc3e5d06b5a38",
        ),
        "linux_arm": (
            "go1.15beta1.linux-armv6l.tar.gz",
            "d4da5c06097be8d14aeeb45bf8440a05c82e93e6de26063a147a31ed1d901ebc",
        ),
        "linux_ppc64le": (
            "go1.15beta1.linux-ppc64le.tar.gz",
            "33f7bed5ee9d4a0343dc90a5aa4ec7a1db755d0749b624618c15178fd8df4420",
        ),
        "linux_s390x": (
            "go1.15beta1.linux-s390x.tar.gz",
            "493b4449e68d0deba559e3f23f611310467e4c70d30b3605ff06852f14477457",
        ),
        "windows_386": (
            "go1.15beta1.windows-386.zip",
            "6ef5301bf03a298a023449835a941d53bf0830021d86aa52a5f892def6356b19",
        ),
        "windows_amd64": (
            "go1.15beta1.windows-amd64.zip",
            "072c7d6a059f76503a2533a20755dddbda58b5053c160cb900271bb039537f88",
        ),
    },
}
