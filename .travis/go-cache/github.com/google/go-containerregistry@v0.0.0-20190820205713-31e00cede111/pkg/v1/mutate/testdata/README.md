# whiteout\_image.tar

Including whiteout files in our source caused [issues](https://github.com/google/go-containerregistry/issues/305)
when cloning this repo inside a docker build. Removing the whiteout file from
this test data doesn't break anything (since we checked in the tar), but if you
want to rebuild it for some reason:

```
touch whiteout/.wh.foo.txt
```
