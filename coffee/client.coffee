# Public Domain (-) 2011-2013 The Espra Authors.
# See the Espra UNLICENSE file for details.

define 'espra', (exports, root) ->

  doc = root.document
  doc.$ = doc.getElementById
  body = doc.body
  domly = exports.domly
  local = root.localStorage 
  local['auth'] = 1

  # Create the root #body container element.
  exports.container = container = doc.createElement 'div'
  container.id = 'body'
  container.style.display = 'none'
  body.appendChild container

  $focus = null
  focus = exports.focus = ->
    if $focus
      body.style.height = '100%'
      body.className = 'bg'
      container.style.display = 'block'
      $focus.focus()
    return

  handleLogin = (e) ->
    e.preventDefault()
    e.stopPropagation()
    user = doc.$ 'login-user'
    return

  handlePersistLogin = () ->
    if this.checked
      local['login.persist'] = '1'
    else
      local.removeItem 'login.persist'
    return

  handleSignup = (e) ->
    e.preventDefault()
    e.stopPropagation()
    return

  showPowerSignup = (e) ->
    l = doc.$ 'signup-more-row'
    l.style.display = 'none'
    elems = Array::slice.call doc.getElementsByClassName 'signup-hidden'
    for elem in elems
      elem.className = ''
    e.preventDefault()
    l = doc.$ 'signup-nick'
    l.focus()

  nickChanged = false
  onSignupName = () ->
    if !nickChanged
      l = doc.$ 'signup-nick'
      l.value = this.value.split(' ')[0].toLowerCase()

  onSignupNick = () ->
    nickChanged = true

  exports.showHomeScreen = showHomeScreen = () ->
    validated = false
    loginPrev = local['login.prev'] or ''
    checkbox = id: 'login-persist', onclick: handlePersistLogin, tabindex: 3, type: 'checkbox'
    if local['login.persist']
      checkbox.checked = 'checked'
    data = [
      'div', id: 'home',
        ['a', href: '/', id: 'espra', 'Espra'],
        ['form', id: 'login', onsubmit: handleLogin,
          ['table', id: 'login-table',
            ['tr',
              ['td', ['label', for: 'login-user', 'Email']],
              ['td', ['label', for: 'login-pass', 'Passphrase']]
            ],
            ['tr',
              ['td', ['input', id: 'login-user', tabindex: 1, value: loginPrev]],
              ['td', ['input', id: 'login-pass', tabindex: 2, type: 'password']]
            ],
            ['tr', id: 'login-custom',
              ['td', ['input', checkbox], ['label', for: 'login-persist', id: 'login-persist-label', 'Keep me logged in']],
              ['td', ['input', id: 'login-submit', tabindex: 4, type: 'submit', value: 'Login!']],
            ]
          ],
          ['hr', class: 'clear']
        ],
        ['form', id: 'signup', onsubmit: handleSignup,
          ['div', id: 'signup-title', 'Sign up!'],
          ['table', id: 'signup-table',
            ['tr',
              ['td', class: 'signup-label', ['label', for: 'signup-name', 'Full Name *']],
              ['td', ['input', id: 'signup-name', onchange: onSignupName, tabindex: 11, '']],
            ],
            ['tr',
              ['td', ''],
              ['td', ['div', id: 'signup-name-i', '']],
            ],
            ['tr',
              ['td', class: 'signup-label', ['label', for: 'signup-email', 'Email *']],
              ['td', ['input', id: 'signup-email', tabindex: 12, '']],
            ],
            ['tr',
              ['td', ''],
              ['td', ['div', id: 'signup-email-i', '']],
            ],
            ['tr',
              ['td', class: 'signup-label', ['label', for: 'signup-pass', 'Passphrase *']],
              ['td', ['input', id: 'signup-pass', tabindex: 13, type: 'password', '']],
            ],
            ['tr',
              ['td', ''],
              ['td', ['div', id: 'signup-pass-i', '']],
            ],
            ['tr',
              ['td', class: 'signup-label', ['label', for: 'signup-gender', 'Gender']],
              ['td', ['select', id: 'signup-gender', tabindex: 14,
                ['option', value: '-', ''], ['option', value: 'M', 'Male'], ['option', value: 'F', 'Female'], ['option', value: 'C', "It's complicated"]]],
            ],
            ['tr', id: 'signup-more-row',
              ['td', ''],
              ['td', ['a', href: '', id: 'signup-more', onclick: showPowerSignup, tabindex: 15, 'Show Power User Options']],
            ],
            ['tr', class: 'signup-hidden',
              ['td', class: 'signup-label', ['label', for: 'signup-nick', 'Nick']],
              ['td', ['input', id: 'signup-nick', onchange: onSignupNick, tabindex: 16, '']],
            ],
            ['tr', class: 'signup-hidden',
              ['td', class: 'signup-label', ['label', for: 'signup-id', 'ID Number']],
              ['td', ['input', id: 'signup-id', tabindex: 17, '']],
            ],
            ['tr',
              ['td', ''],
              ['td',
                ['input', id: 'signup-terms', tabindex: 18, type: 'checkbox', value: 'agree'],
                ['label', id: 'signup-terms-label', for: 'signup-terms', 'I accept the Espra terms of service']],
            ],
          ],
          ['input', id: 'signup-submit', tabindex: 19, type: 'submit', value: 'Join Espra!']
          ['hr', class: 'clear']
        ],
    ]

    domly data, container
    $focus = doc.$ 'login-user'

  exports.run = (incr) ->

    incr()
    if !exports.initNotifi(container)
      alert("Sorry, your browser doesn't seem to support CSS Transitions.")
      throw 'wrench'

    if local['auth'] isnt '1'
      showHomeScreen()
    else
      # setup client master view logic
      # load  resource specific modules including compiled templates, view code, sass and data APIs as ASSETS, 
      # then load resource specifc data and bind to modules
      # load_modules {MODULES}
      init.JS "#{$STATIC}#{ASSETS['test.js']}", ->
        test.run()
    return
  return
