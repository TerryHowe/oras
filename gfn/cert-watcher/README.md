<!--
SPDX-FileCopyrightText: Copyright (c) NVIDIA CORPORATION & AFFILIATES. All rights reserved.
SPDX-License-Identifier: Apache-2.0

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->
# Cert Watcher

This program watches for changes to SECRETS_FILE json file, parses the value at jsonPath CA_CHAIN_PATH, CERT_PATH & KEY_PATH and writes the full CA Chain (Certificate + CA_Chain) output to CERT_OUTPUT_FILE and key output to KEY_OUTPUT_FILE. The program is also triggered at least once every 15 minutes to catch issues with file notifications.

Config via Env Var
- CA_CHAIN_PATH
- CERT_PATH
- KEY_PATH
- CERT_OUTPUT_FILE
- KEY_OUTPUT_FILE
- SECRETS_FILE
