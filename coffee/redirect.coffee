# Public Domain (-) 2012 The Espra Authors.
# See the Espra UNLICENSE file for details.

# Compile this with:
# coffee -b -c redirect.coffee && uglifyjs2 redirect.js -m -r 'window' | cut -d';' -f 3- | sed 's/\(.*\)./\1/'

redirect = () ->
  location = window.location
  hash = location.hash
  if hash.length < 2 or (url = decodeURIComponent(hash.slice(1))) is "_private"
    location.replace(U)
  else
    location.replace(url)
  return
