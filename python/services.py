# Public Domain (-) 2008-2012 The Espra Authors.
# See the Espra UNLICENSE file for details.

"""Proto Espra Python Services."""

import logging
import os
import sys

from cgi import FieldStorage
from json import dumps as encode_json
from os.path import dirname
from traceback import format_exception
from urllib import unquote as urlunquote

from google.appengine.runtime.apiproxy_errors import CapabilityDisabledError

# Extend the sys.path to include the parent and ``lib`` sibling directories.
sys.path.insert(0, dirname(__file__))
sys.path.insert(0, 'lib')

from pygments import highlight
from pygments.formatters import HtmlFormatter
from pygments.lexers import get_lexer_by_name, TextLexer

from webob import Request as WebObRequest # this import patches cgi.FieldStorage
                                          # to behave better for us too!

from config import SECRET_KEY

# ------------------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------------------

HTML_RESPONSE = """<!DOCTYPE html>
<meta charset=utf-8>
<title>%(msg)s</title>
<link href='//fonts.googleapis.com/css?family=Droid+Sans' rel=stylesheet>
<style>
body {
  font-family: 'Droid Sans', Verdana, sans-serif;
  font-size: 40px;
  padding: 10px 7px;
}
</style>
<body>
%(msg)s
"""

ERROR_401 = HTML_RESPONSE % dict(msg="401 Not Authorized")
ERROR_404 = HTML_RESPONSE % dict(msg="404 Not Found")
ERROR_503 = HTML_RESPONSE % dict(msg="503 Service Unavailable")
ROOT_HTML = HTML_RESPONSE % dict(msg="Python API Endpoint")

RESPONSE_HEADERS_HTML = [("Content-Type", "text/html; charset=utf-8")]
RESPONSE_OPT = ("200 OK", [("Allow:", "OPTIONS, GET, HEAD, POST")])
RESPONSE_200 = ("200 OK", RESPONSE_HEADERS_HTML)
RESPONSE_401 = ("401 Unauthorized", RESPONSE_HEADERS_HTML +
                [("WWW-Authenticate", "Token realm='Service', error='token_expired'")])
RESPONSE_404 = ("404 Not Found", RESPONSE_HEADERS_HTML)
RESPONSE_501 = ("501 Not Implemented", [])
RESPONSE_503 = ("503 Service Unavailable", RESPONSE_HEADERS_HTML)

SUPPORTED_HTTP_METHODS = frozenset(['GET', 'HEAD', 'POST'])
VALID_REQUEST_CONTENT_TYPES = frozenset([
    '', 'application/x-www-form-urlencoded', 'multipart/form-data'
    ])

# ------------------------------------------------------------------------------
# Service Utilities
# ------------------------------------------------------------------------------

SERVICE_REGISTRY = {}

# The ``service`` decorator is used to turn a function into a service.
def service(name, cache=False):
    def __register_service(function):
        SERVICE_REGISTRY[name] = (function, cache)
        return function
    return __register_service

# ------------------------------------------------------------------------------
# App Runner
# ------------------------------------------------------------------------------

def app(
    env, start_response, dict=dict, isinstance=isinstance, ord=ord,
    in_production=os.environ.get('SERVER_SOFTWARE', '').startswith('Google'),
    unicode=unicode, urlunquote=urlunquote,
    ):

    http_method = env['REQUEST_METHOD']
    def respond(prelude, content=None):
        if http_method == 'HEAD':
            if content:
                headers = prelude[1] + [("Content-Length", str(len(content)))]
                start_response(prelude[0], headers)
            else:
                start_response(*prelude)
            return []
        start_response(*prelude)
        return [content]

    if http_method == 'OPTIONS':
        return respond(RESPONSE_OPT)

    if http_method not in SUPPORTED_HTTP_METHODS:
        return respond(RESPONSE_501)

    if in_production and env['wsgi.url_scheme'] != 'https':
        return respond(RESPONSE_401, ERROR_401)

    _path_info = env['PATH_INFO']
    if isinstance(_path_info, unicode):
        args = [arg for arg in _path_info.split(u'/') if arg]
    else:
        args = [
            unicode(arg, 'utf-8', 'strict')
            for arg in _path_info.split('/') if arg
            ]

    if not args:
        return respond(RESPONSE_200, ROOT_HTML)

    service = args[0]
    args = args[1:]

    if service not in SERVICE_REGISTRY:
        return respond(RESPONSE_404, ERROR_404)

    service, cache = SERVICE_REGISTRY[service]
    kwargs = {}

    for part in [
        sub_part
        for part in env['QUERY_STRING'].lstrip('?').split('&')
        for sub_part in part.split(';')
        ]:
        if not part:
            continue
        part = part.split('=', 1)
        if len(part) == 1:
            value = None
        else:
            value = part[1]
        key = urlunquote(part[0].replace('+', ' '))
        if value:
            value = unicode(
                urlunquote(value.replace('+', ' ')), 'utf-8', 'strict'
                )
        else:
            value = None
        if key in kwargs:
            _val = kwargs[key]
            if isinstance(_val, list):
                _val.append(value)
            else:
                kwargs[key] = [_val, value]
            continue
        kwargs[key] = value

    # Parse the POST body if it exists and is of a known content type.
    if http_method == 'POST':

        content_type = env.get('CONTENT-TYPE', '')
        if ';' in content_type:
            content_type = content_type.split(';', 1)[0]

        if content_type in VALID_REQUEST_CONTENT_TYPES:
            post_environ = env.copy()
            post_environ['QUERY_STRING'] = ''
            post_data = FieldStorage(
                environ=post_environ, fp=env['wsgi.input'],
                keep_blank_values=True
                ).list or []
            for field in post_data:
                key = field.name
                if field.filename:
                    value = field
                else:
                    value = unicode(field.value, 'utf-8', 'strict')
                if key in kwargs:
                    _val = kwargs[key]
                    if isinstance(_val, list):
                        _val.append(value)
                    else:
                        kwargs[key] = [_val, value]
                    continue
                kwargs[key] = value

    auth = kwargs.pop('__auth__', None)
    if not auth or len(auth) != len(SECRET_KEY):
        return respond(RESPONSE_401, ERROR_401)

    total = 0
    for x, y in zip(auth, SECRET_KEY):
        total |= ord(x) ^ ord(y)
    if total != 0:
        return respond(RESPONSE_401, ERROR_401)

    try:
        content = dict(response=service(*args, **kwargs))
    except CapabilityDisabledError:
        return respond(RESPONSE_503, ERROR_503)
    except Exception, error:
        logging.critical(''.join(format_exception(*sys.exc_info())))
        content = dict(
            error=("%s: %s" % (error.__class__.__name__, error))
            )

    content = encode_json(content)
    headers = [
        ("Content-Type", "application/json; charset=utf-8"),
        ("Content-Length", str(len(content)))
    ]

    if cache and http_method == 'GET':
        if isinstance(cache, int):
            duration = cache
        else:
            duration = 86400
        headers.append(("Cache-Control", "public, max-age=%d;" % duration))

    start_response("200 OK", headers)
    if http_method == 'HEAD':
        return []
    return [content]

# -----------------------------------------------------------------------------
# Services
# -----------------------------------------------------------------------------

@service('hilite')
def hilite(text, lang=None):
    if lang:
        try:
            lexer = get_lexer_by_name(lang)
        except ValueError:
            lang = 'txt'
            lexer = TextLexer()
    else:
        lang = 'txt'
        lexer = TextLexer()
    formatter = HtmlFormatter(
        cssclass='syntax %s' % lang, lineseparator='<br/>'
        )
    return highlight(text, lexer, formatter)
