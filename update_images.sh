#!/bin/bash

set -o errexit
set -o nounset

IMGS="postgresql-operator backup-manager service-binding-controller"
for IMG in $IMGS
do
    # TODO: Switch to "sed" when copying inside github action
    # This regex could be stricter. For example, it doesn't check that in the semver version there are no leading zeros
    # if the semver number has more than one digit (e.g. v01.1.1 would pass). But right now it's enough and it's much
    # more readable this way than in "stricter" formats.
    IMG_FIELD_REGEXP="s/^[[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$IMG:v\([[:digit:]]\{1,\}.[[:digit:]]\{1,\}.[[:digit:]]\{1,\}\)\"\{0,1\}/\1/p"
    CURRENT_VERSION=$(gsed -n $IMG_FIELD_REGEXP "deploy/a8s/$IMG.yaml")
    NEW_VERSION="11.53.22"
    # This check might misbehave if one of the semver versions has a version number with leading zeros and more than one
    # digit. But that should never happen since we control those version numbers and there's no reason why we should 
    # end up having leading zeros in version numbers with more than one digit.
    if [[ "$NEW_VERSION" > "$CURRENT_VERSION" ]]
    then
        UPDATE_IMG_FIELD_REGEXP="s/^\([[:space:]]\{1,\}image:[[:space:]].\{1,\}\/$IMG:v\)[[:digit:]]\{1,\}.[[:digit:]]\{1,\}.[[:digit:]]\{1,\}\(\"\{0,1\}\)/\1$NEW_VERSION\2/"
        gsed -i $UPDATE_IMG_FIELD_REGEXP "deploy/a8s/$IMG.yaml"
        git add "deploy/a8s/$IMG.yaml"
        git commit -m "Bump $IMG to v$NEW_VERSION"
    else
        echo "$IMG current version in a8s is v$CURRENT_VERSION, most recent version found is v$NEW_VERSION, no update needed"
    fi
done

# TODO: Handle logging
# TODO: Handle opensearch-dashboards
