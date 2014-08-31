FILES="$(git ls-files | grep '.go')"
for file in $FILES; do
  gofmt -w $file
done
