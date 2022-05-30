#!/bin/bash -eu

# # echo a
# # echo b
# # exec echo c
# # exec echo d

# declare -p
# source /tmp/.monkey_5916184413087258808_00000000000000000005.env >/dev/null 2>&1
# set -o errexit
# set -o errtrace
# set -o nounset
# set -o pipefail
# set -o xtrace

#     ./vault/bin/vault server -dev -dev-root-token-id='root' -address='http://127.0.0.1:8200' -exit-on-core-shutdown &
#     # export vault_pid=$!
#     vault_pid=$!

#     # Wait until server is up
#     until curl --output /dev/null --silent --fail -H 'X-Vault-Token: root' http://127.0.0.1:8200/v1/sys/internal/specs/openapi; do
#         sleep .1
#     done
    
# set +o xtrace
# set +o pipefail
# set +o nounset
# set +o errtrace
# set +o errexit
# declare -p > /tmp/.monkey_5916184413087258808_00000000000000000005.env


set -o errexit
set -o errtrace
set -o nounset
set -o pipefail

# master <- master$UUID.script
# slave <- slave$UUID.script
# ltime <- get master mtime
# loop
#   mtime <- get master mtime
#   if deleted
#     break
#   if mtime > ltime
#     ltime <- mtime
#     set -o xtrace
#     source $(cat master)
#     set +o xtrace
#     touch slave
#   sleep .5  TODO: don't poll



# while :; do
for i in $(seq 1 3); do
	# source /tmp/.monkey_5916184413087258808_00000000000000000005_vault_dev.script
	# source <(
	# 	set -o xtrace
	# 	echo 'echo a'
	# )

	set -o xtrace

	source a.sh
	source <(
		echo 'source b.sh'
	)
	source c.sh

	set +o xtrace
	break
done

# ..._$UUID.script
# i <- ..._$UUID.script.i
# o <- ..._$UUID.script.o
while :; do
	if ! script=$(cat "$i"); then
		# File was deleted
		break
	fi

	if [[ -z "$script" ]]; then
		sleep 1
		continue
	fi

	set -o xtrace
	source "$script"
	set +o xtrace
	touch "$o"
done
rm -f "$i" "$o" ...self...
# Go side: https://stackoverflow.com/a/21508289
# or https://github.com/fsnotify/fsnotify

create loop script from template with hard paths
spawn loop in background
