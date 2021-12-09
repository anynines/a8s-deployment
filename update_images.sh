#!/bin/bash

# TODO: Switch to "sed" when copying inside github action

set -o errexit
set -o nounset
set -o pipefail

# Right now the core components are the PostgreSQL Operator, the Backup Manager and the Service
# Binding Controller.
update_core_component_img_and_commit () {
    local component=$1
    local new_version=$2
    local manifest="deploy/a8s/$component.yaml"

    local get_version_sed_cmd="s/^[[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$component:\(v[\.[:digit:]]\{1,\}\)\"\{0,1\}$/\1/p"
    local current_version=$(gsed -n $get_version_sed_cmd $manifest)

    if [[ "$new_version" > "$current_version" ]]
    then
        local update_version_sed_cmd="s/^\([[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$component:\)v[\.[:digit:]]\{1,\}\(\"\{0,1\}\)$/\1$new_version\2/"
        gsed -i "$update_version_sed_cmd" "$manifest"
        # TODO: Uncomment before pushing real version.
        # git add "$manifest"
        # git commit -m "Bump $img to $new_version"
    else
        echo "$component current version is $current_version, most recent version found is $new_version, no update needed"
    fi
}

VERSIONED_IMGS="postgresql-operator:v0.9.0 backup-manager:v0.7.0 service-binding-controller:v0.5.0"
for VERSIONED_IMG in $VERSIONED_IMGS
do
    # Extract image name and version as separate variables
    IMG=$(echo $VERSIONED_IMG | cut -d ':' -f 1)
    NEW_VERSION=$(echo $VERSIONED_IMG | cut -d ':' -f 2)

    # Set variables that depend on the specific image that we're considering
    if [[ "$IMG" == "fluentd" ]]
    then
        echo "fluentd"
    elif [[ "$IMG" == "opensearch-dashboards" ]]
    then
        echo "opensearch-dashboards"
    else
        update_core_component_img_and_commit $IMG $NEW_VERSION
    fi
done

# TODO: Handle logging
# TODO: Handle opensearch-dashboards
# TODO: Refactor to reuse the function
