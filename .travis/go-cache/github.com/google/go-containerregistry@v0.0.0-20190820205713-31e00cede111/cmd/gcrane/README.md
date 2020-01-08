# gcrane

This tool implements a superset of the commands under [`cmd/crane`](), with
additional commands that are specific to [gcr.io](https://gcr.io).

Note that this relies on some implementation details of GCR that are not
consistent with the [registry spec](https://docs.docker.com/registry/spec/api/),
so this may break in the future.

## ls

`gcrane ls` exposes a more complex form of `ls` than `crane`, which allows for
listing tags, manifests, and sub-repositories.

## cp

`gcrane cp` supports a `-r` flag that copies images recursively, which is useful
for backing up images, georeplicating images, or renaming images en masse.

## gc

`gcrane gc` will calculate images that can be garbage-collected.
By default, it will print any images that do not have tags pointing to them.

This can be composed with `gcrane delete` to actually garbage collect them:
```shell
gcrane gc gcr.io/${PROJECT_ID}/repo | xargs -n1 gcrane delete
```

<!--
TODO: implement this.

## untag

The [registry api](https://docs.docker.com/registry/spec/api/#deleting-an-image)
only allows deleting images by digest:

> For deletes, reference must be a digest or the delete will fail.

gcr.io allows deleting a manifest with a *tag* reference, which it
interprets as a request to untag the image, not delete it. This leaves the
image intact but still pullable by digest (or any other tags).
-->
