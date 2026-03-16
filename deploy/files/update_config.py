# SPDX-FileCopyrightText: Copyright (c) 2023-2025 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
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

"""
Update containerd config.toml to set the registry config_path for certs.d
mirror configuration.

Safety properties:
  - Idempotent: skips if config_path is already set correctly.
  - Creates missing registry section instead of erroring out.
  - Backs up to .bak only on first run (preserves the pristine AMI copy).
  - Copies (not renames) the original file to .bak before writing.
  - Restores from backup if the write fails.

The .bak file is the "master copy" of the original config.toml as it
existed before nvcf-container-cache ever touched it. The rollback tool
(remove-containerd-configuration.yaml) restores from this file.

This avoids the race condition with nvidia-container-toolkit documented
in DGXCINC-3249 / NVBug 5979011 where the old script would rename
config.toml to .bak before validation, and on failure leave no
config.toml on disk.
"""

import os
import shutil
import sys

import toml

DESIRED_CONFIG_PATH = "/etc/containerd/certs.d"


def update_containerd_config(config_file):
    if not os.path.exists(config_file):
        print(f"Warning: {config_file} not found, skipping config update")
        return

    with open(config_file, "r") as f:
        config = toml.load(f)

    plugins = config.setdefault("plugins", {})
    cri = plugins.setdefault("io.containerd.grpc.v1.cri", {})
    registry = cri.setdefault("registry", {})

    if registry.get("config_path") == DESIRED_CONFIG_PATH:
        print(f"config_path already set in {config_file}, nothing to do.")
        return

    registry.pop("mirrors", None)
    registry["config_path"] = DESIRED_CONFIG_PATH

    backup_file = config_file + ".bak"
    if not os.path.exists(backup_file):
        shutil.copy2(config_file, backup_file)
        print(f"Created pristine backup: {backup_file}")

    try:
        with open(config_file, "w") as f:
            toml.dump(config, f)
        print(f"Updated config_path in {config_file} successfully.")
    except Exception as e:
        print(f"Error writing config, restoring backup: {e}")
        shutil.copy2(backup_file, config_file)
        raise


if __name__ == "__main__":
    config_path = sys.argv[1] if len(sys.argv) > 1 else "/etc/containerd/config.toml"

    try:
        update_containerd_config(config_path)
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)
