# Public Domain (-) 2011-2013 The Espra Authors.
# See the Espra UNLICENSE file for details.

define 'espra', (exports, root) ->

  doc = root.document
  events = {}
  evid = 1
  isArray = Array.isArray
  lastid = 1
  watchList = {}
  
  processTagName = (tag, elem) ->
    # what if a #id comes after a .class in the tagName
    split = tag.split '.'
    if split.length > 1
      [tag, classes...] = split
      classes = classes.join ' '
    else
      classes = null

    split = tag.split '#'
    if split.length is 2
      [tag, id] = split
    else
      id = null
      #if setID and not elem.id
      #  id = "$#{lastid++}"

    if classes
      if elem.className.length > 0
        elem.className += " #{classes}"
      else
        elem.className = classes

    if id
      elem.id = id
        
    return

  processAttrs = (elem, attrs) ->
    console.log k + " : " + v
    for k, v of attrs
      if k.lastIndexOf('on', 0) is 0
        if typeof v isnt 'function'
          continue
        if !elem.__evi
          elem.__evi = evid++
        type = k.slice 2
        if events[elem.__evi]
          events[elem.__evi].push [type, v, false]
        else
          events[elem.__evi] = [[type, v, false]]
        elem.addEventListener type, v, false
      else
        elem[propFix[k] or k] = v

  evalExpr = (expr) ->
    localBindings = []
    start = 0
    kwargs = expr[0]
    if !isArray kwargs and typeof kwargs is 'object'
      start = 1

    for term in expr[start...expr.length]
      if term[0] == 3  # var/func
        # resolve var in local namespace
        # if its a function inspect its signature
        if isfunction term[1]
          # if start == 1 then match kwargs to signature
        else
          # return its value
      else if term[0] == 10  # builtin
        # resolve var from a list of builtins
      else if term[0] == 6 # integer
      else if term[0] == 5 # string
      else if term[0] == 'expr'
        bind_res = evalExpr(term[1...expr.length])

    for b in localBindings
      if watchList[b] == undefined
        watchList[b] = []
      watchList[b][watchList[b].length][expr]
    
    return val
      
  buildExpr = (data, parent) ->
    
    return [val, bindings]

  isTruish = (val) ->
    if isBool(val)
      if val == true
        return true
      else
        return false

  
  buildFor = (for_expr, in_expr, repeatedTree, parent) ->
    for item in in_expr
      
      buildTree repeatedTree parent
    return
    
  buildIf = (expr, condSubTree, parent) ->
    # bind input params and output domly
    [val, val_ref] = evalExpr expr
    if isTruish(val)
      buildTree condSubTree, parent
    

  buildTree = (data, parent) ->
    # build a tree down from a single node
    tag = data[0]
    l = data.length
    if tag == 'expr'
      buildExpr data[1...l]
    else if typeof tag == 'number' && tag % 1 == 0
      if tag == 17
        buildIf data[1]['if'], data[2...l], parent
      if tag == 19
        buildFor data[1]['for'], data[1]['in'], data[2...1], parent
      console.log "opcode: " + tag      
    else
      console.log tag
      elem = doc.createElement tag

      parent.appendChild elem
      if l > 1
        start = 1
        if !isArray attrs and typeof attrs is 'object'
          processAttrs elem attrs
          start = 2
      for child in data[start...l]
        if typeof child is 'string'
          elem.appendChild document.createTextNode child
        else
          buildTree child, elem
      
  buildDOM = (data, parent, setID) ->
    if isArray data[0]
      # there is more than one element at the top level
      for elem in data
        if typeof elem is 'string'
          parent.appendChild document.createTextNode elem
        else
          buildTree elem, parent
    else
      buildTree data, parent
    return
        
  exports.domly = (data, target, retElem) ->
    frag = doc.createDocumentFragment()
    if retElem
      id = buildDOM data, frag, true
      target.appendChild frag
      return doc.getElementById id
    buildDOM data, frag, false
    target.appendChild frag
    return

  purgeDOM = (elem) ->
    evi = elem.__evi
    if evi
      for [type, func, capture] in events[evi]
        elem.removeEventListener type, func, capture
      delete events[evi]
    children = elem.childNodes
    if children
      for child in children
        purgeDOM child
    return

  exports.rmtree = (parent, elem) ->
    parent.removeChild elem
    purgeDOM elem
    return

  return
