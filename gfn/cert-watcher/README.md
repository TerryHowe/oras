# Cert Watcher

This program watches for changes to SECRETS_FILE json file, parses the value at jsonPath CA_CHAIN_PATH, CERT_PATH & KEY_PATH and writes the full CA Chain (Certificate + CA_Chain) output to CERT_OUTPUT_FILE and key output to KEY_OUTPUT_FILE. The program is also triggered at least once every 15 minutes to catch issues with file notifications.

Config via Env Var
- CA_CHAIN_PATH
- CERT_PATH
- KEY_PATH
- CERT_OUTPUT_FILE
- KEY_OUTPUT_FILE
- SECRETS_FILE
