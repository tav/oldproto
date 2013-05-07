# Public Domain (-) 2012-2013 The Espra Authors.
# See the Espra UNLICENSE file for details.

define 'espra', (exports, root) ->

  doc = root.document
  doc.$e = doc.createElement
  $notifi = doc.$e 'div'
  $notifi.id = 'notifi'

  transitionEvents =
    Moz: 'transitionend'
    ms: 'MSTransitionEnd'
    O: 'oTransitionEnd'
    Webkit: 'webkitTransitionEnd'

  transitionEvent = 'transitionend'
  getEvent = (style) ->
    if typeof style['TransitionProperty'] is 'string'
      return 1
    for prefix, event of transitionEvents
      if typeof style[prefix+'TransitionProperty'] is 'string'
        transitionEvent = event
        return 1
    return

  exports.initNotifi = (elem) ->
    if !getEvent(elem.style)
      return
    return 1
