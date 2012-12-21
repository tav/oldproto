# Public Domain (-) 2008-2012 The Espra Authors.
# See the Espra UNLICENSE file for details.

"""A sexy micro-framework for use with Google App Engine."""

import logging
import os
import sys

from BaseHTTPServer import BaseHTTPRequestHandler
from binascii import hexlify
from cgi import FieldStorage
from cStringIO import StringIO
from datetime import datetime
from hashlib import sha1
from json import dumps as json_encode, loads as json_decode
from md5 import md5
from os import urandom
from os.path import dirname, exists, join as join_path, getmtime
from threading import local
from traceback import format_exception
from urllib import quote as urlquote, unquote as urlunquote
from urlparse import urljoin
from wsgiref.headers import Headers

from google.appengine.ext.blobstore import parse_blob_info
from google.appengine.runtime.apiproxy_errors import CapabilityDisabledError

# Extend the sys.path to include the parent and ``lib`` sibling directories.
sys.path.insert(0, dirname(__file__))
sys.path.insert(0, 'lib')

from cookie import SimpleCookie # note: this is our cookie and not Cookie...
from exception import html_format_exception
from mako.exceptions import RichTraceback
from mako.template import Template as MakoTemplate
from tavutil.crypto import (
    create_tamper_proof_string, secure_string_comparison,
    validate_tamper_proof_string
    )

from tavutil.jsonp import is_valid_jsonp_callback_value

from webob import Request as WebObRequest # this import patches cgi.FieldStorage
                                          # to behave better for us too!

from config import (
    DEBUG, SECURE_COOKIE_DURATION, SECURE_COOKIE_KEY, SKINS_ENABLED,
    STATIC_HTTP_HOSTS, STATIC_HTTPS_HOSTS, STATIC_PATH
    )

# ------------------------------------------------------------------------------
# Utility File Reader
# ------------------------------------------------------------------------------

def read(file):
    f = open(file, 'rb')
    data = f.read()
    f.close()
    return data

# ------------------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------------------

ASSETS = json_decode(read('assets.json'))

COOKIE_KEY_NAMES = frozenset([
    'domain', 'expires', 'httponly', 'max-age', 'path', 'secure', 'version'
    ])

SITE_CSS_PATH = str("%s%s" % (STATIC_PATH, ASSETS['site.data.css']))
SITE_CSS_IE_PATH = str("%s%s" % (STATIC_PATH, ASSETS['site.css']))

ERROR_WRAPPER = """<!DOCTYPE html>
<meta charset=utf-8>
<title>Error!</title>
<!--[if !IE]><!-->
  <link rel=stylesheet href=%s>
<!--<![endif]-->
<!--[if gte IE 8]>
  <link rel=stylesheet href=%s>
<![endif]-->
<!--[if lte IE 7]>
  <link rel=stylesheet href=%s>
<![endif]-->
<body>
%%s
""" % (SITE_CSS_PATH, SITE_CSS_PATH, SITE_CSS_IE_PATH)

ERROR_401 = ERROR_WRAPPER % """
  <div class="site-error">
    <h1>Not Authorized</h1>
    Your session may have expired or you may not have access.
    <ul>
      <li><a href="/">Return home</a></li>
      <li><a href="/login">Login</a></li>
    </ul>
  </div>
  """ # emacs"

ERROR_404 = ERROR_WRAPPER % """
  <div class="site-error">
    <h1>The item you requested was not found</h1>
    You may have clicked a dead link or mistyped the address. Some web addresses
    are case sensitive.
    <ul>
      <li><a href="/">Return home</a></li>
    </ul>
  </div>
  """ # emacs"

ERROR_500_BASE = ERROR_WRAPPER % """
  <div class="site-error">
    <h1>Sorry, something went wrong!</h1>
    There was an application error. This has been logged and will be resolved as
    soon as possible.
    <ul>
      <li><a href="/">Return home</a></li>
    </ul>
    %s
  </div>
  """ # emacs"

ERROR_500 = ERROR_500_BASE % ""

ERROR_500_TRACEBACK = ERROR_500_BASE % """
    <div class="traceback">%s</div>
  """ # emacs"

ERROR_503 = ERROR_WRAPPER % """
  <div class="site-error">
    <h1>Service Unavailable</h1>
    Google App Engine is currently down for a scheduled maintenance.
    Please try again later.
    <ul>
      <li><a href="/">Return home</a></li>
    </ul>
  </div>
  """ # emacs"

HTTP_STATUS_MESSAGES = BaseHTTPRequestHandler.responses

RESPONSE_NOT_IMPLEMENTED = ("501 Not Implemented", [])

RESPONSE_OPTIONS = (
    "200 OK",
    [("Allow:", "OPTIONS, GET, HEAD, POST")]
    )

RESPONSE_HEADERS_HTML = [
    ("Content-Type", "text/html; charset=utf-8")
]

STATUS_301 = "301 Moved Permanently"
STATUS_302 = "302 Found"

RESPONSE_401 = ("401 Unauthorized", RESPONSE_HEADERS_HTML +
                [("WWW-Authenticate", "Token realm='Service', error='token_expired'")])

RESPONSE_404 = ("404 Not Found", RESPONSE_HEADERS_HTML)
RESPONSE_500 = ("500 Server Error", RESPONSE_HEADERS_HTML)
RESPONSE_503 = ("503 Service Unavailable", RESPONSE_HEADERS_HTML)

RESPONSE_JSON_ERROR = (
    "500 Server Error",
    [("Content-Type", "application/json; charset=utf-8")]
    )

if os.environ.get('SERVER_SOFTWARE', '').startswith('Google'):
    RUNNING_ON_GOOGLE_SERVERS = True
else:
    RUNNING_ON_GOOGLE_SERVERS = False

SECURE_PREFIX = hexlify(urandom(18))

SERVICE_REGISTRY = {}
SUPPORTED_HTTP_METHODS = frozenset(['GET', 'HEAD', 'POST'])

VALID_REQUEST_CONTENT_TYPES = frozenset([
    '', 'application/x-www-form-urlencoded', 'multipart/form-data'
    ])

# ------------------------------------------------------------------------------
# Exceptions
# ------------------------------------------------------------------------------

# Services can throw exceptions to return specifc HTTP response codes.
#
# All the errors subclass the ``BaseHTTPError``.
class BaseHTTPError(Exception):
    pass

# The ``Redirect`` exception is used to handle HTTP 301/302 redirects.
class Redirect(BaseHTTPError):
    def __init__(self, uri, permanent=False):
        self.uri = urljoin('', str(uri))
        self.permanent = permanent

# The ``HTTPContent`` is used to return the associated content.
class HTTPContent(BaseHTTPError):
    def __init__(self, content):
        self.content = content

# The ``AuthError`` is used to represent the 401 Not Authorized error.
class AuthError(BaseHTTPError):
    pass

# The ``NotFound`` is used to represent the classic 404 error.
class NotFound(BaseHTTPError):
    pass

# The ``HTTPError`` is used to represent all other response codes.
class HTTPError(BaseHTTPError):
    def __init__(self, code=500):
        self.code = code

# ------------------------------------------------------------------------------
# Static
# ------------------------------------------------------------------------------

if RUNNING_ON_GOOGLE_SERVERS:

    def STATIC(
        ctx, path, cache={}, assets=ASSETS, prefix=STATIC_PATH,
        http_hosts=STATIC_HTTP_HOSTS, https_hosts=STATIC_HTTPS_HOSTS,
        l1=len(STATIC_HTTP_HOSTS), l2=len(STATIC_HTTPS_HOSTS)
        ):
        ssl = ctx.ssl_mode
        key = (ssl, path)
        if key in cache:
            return cache[key]
        if ssl:
            hosts, l = http_hosts, l1
        else:
            hosts, l = https_hosts, l2
        return cache.setdefault(key, "//%s%s%s" % (
            hosts[int('0x' + md5(path).hexdigest(), 16) % l], prefix, assets[path]
            ))

else:

    def STATIC(ctx, path, cache={}, prefix=STATIC_PATH):
        if path in cache:
            return cache[path]
        assets = json_decode(read('assets.json'))
        return cache.setdefault(path, "//%s%s%s" % (ctx.host, prefix, assets[path]))

# ------------------------------------------------------------------------------
# Memcache
# ------------------------------------------------------------------------------

# Generate cache key/info for the render service call.
def cache_key_gen(ctx, cache_spec, name, *args, **kwargs):

    user = ''
    if cache_spec.get('user', True):
        user = ctx.username
        if (not cache_spec.get('anon', True)) and not user:
            return

    if cache_spec.get('ignore_args', False):
        args = ()

    if cache_spec.get('ignore_kwargs', False):
        kwargs = {}

    key = sha1(
        "%r-%r-%r" % (user, args, sorted(kwargs.iteritems()))
        ).hexdigest()

    namespace = cache_spec.get('namespace', None)
    if namespace is None:
        namespace = name

    return key, namespace, cache_spec.get('time', 20)

# ------------------------------------------------------------------------------
# Service Utilities
# ------------------------------------------------------------------------------

try:
    from config import SSL_ONLY
except ImportError:
    SSL_ONLY = False

SERVICE_DEFAULT_CONFIG = {
    'admin': False,
    'anon': True,
    'blob': False,
    'cache': False,
    'cache_key': cache_key_gen,
    'cache_spec': dict(namespace=None, time=10, player=True, anon=True),
    'post_encoding': False,
    'skin': SKINS_ENABLED,
    'ssl': SSL_ONLY,
    'xsrf': False
    }

# The ``register_service`` decorator is used to turn a function into a service.
def register_service(name, renderers, **config):
    def __register_service(function):
        __config = SERVICE_DEFAULT_CONFIG.copy()
        __config.update(config)
        for _name in name.split():
            SERVICE_REGISTRY[_name] = (function, renderers, __config)
        return function
    return __register_service

# The default JSON renderer generates JSON-encoded output.
def json(ctx, **content):
    if 'Content-Type' not in ctx.response_headers:
        ctx.response_headers['Content-Type'] = 'application/json; charset=utf-8'
    callback = ctx.json_callback
    if callback:
        if not is_valid_jsonp_callback_value(callback):
            raise ValueError(
                "%r is not an accepted callback parameter." % callback
                )
        return '%s(%s)' % (callback, json_encode(content))
    return json_encode(content)

# ------------------------------------------------------------------------------
# HTTP Utilities
# ------------------------------------------------------------------------------

# Return an HTTP header date/time string.
def get_http_datetime(timestamp=None):
    if timestamp:
        if not isinstance(timestamp, datetime):
            timestamp = datetime.fromtimestamp(timestamp)
    else:
        timestamp = datetime.utcnow()
    return timestamp.strftime('%a, %d %B %Y %H:%M:%S GMT') # %m

# ------------------------------------------------------------------------------
# Null Skin
# ------------------------------------------------------------------------------

class NullSkin(object):
    _ = {}.__getitem__

# ------------------------------------------------------------------------------
# Context
# ------------------------------------------------------------------------------

if SKINS_ENABLED:
    from config import SKIN

# The ``Context`` class encompasses the HTTP request/response. An instance,
# specific to the current request, is passed in as the first parameter to all
# service calls.
class Context(object):

    DEBUG = DEBUG
    STATIC = STATIC

    urlquote = staticmethod(urlquote)
    urlunquote = staticmethod(urlunquote)

    ajax_request = None
    json_callback = None
    end_pipeline = None
    site_host = None

    _cookies_parsed = None
    _xsrf_token = None

    if SKINS_ENABLED:
        skin = SKIN
    else:
        skin = NullSkin()

    skin_dir = None
    skin_id = 'default'

    def __init__(self, service, environ, ssl_mode):
        self.service = service
        self.environ = environ
        self.host = environ['HTTP_HOST']
        self._status = (200, 'OK')
        self._raw_headers = []
        self._response_cookies = {}
        self.response_headers = Headers(self._raw_headers)
        self.ssl_mode = ssl_mode
        if ssl_mode:
            self.scheme = 'https'
        else:
            self.scheme = 'http'

    def set_response_status(self, code, message=None):
        if not message:
            message = HTTP_STATUS_MESSAGES.get(code, ["Server Error"])[0]
        self._status = (code, message)

    def _parse_cookies(self):
        cookies = {}
        cookie_data = self.environ.get('HTTP_COOKIE', '')
        if cookie_data:
            _parsed = SimpleCookie()
            _parsed.load(cookie_data)
            for name in _parsed:
                cookies[name] = _parsed[name].value
        self._request_cookies = cookies
        self._cookies_parsed = 1

    def get_cookie(self, name, default=''):
        if not self._cookies_parsed:
            self._parse_cookies()
        return self._request_cookies.get(name, default)

    def get_secure_cookie(self, name, key=SECURE_COOKIE_KEY, timestamped=True):
        if not self._cookies_parsed:
            self._parse_cookies()
        if name not in self._request_cookies:
            return
        return validate_tamper_proof_string(
            name, self._request_cookies[name], key, timestamped
            )

    def set_cookie(self, name, value, **kwargs):
        cookie = self._response_cookies.setdefault(name, {})
        cookie['value'] = value
        kwargs.setdefault('path', '/')
        if self.ssl_mode:
            kwargs.setdefault('secure', 1)
        for name, value in kwargs.iteritems():
            if value:
                cookie[name.lower()] = value

    def set_secure_cookie(
        self, name, value, key=SECURE_COOKIE_KEY,
        duration=SECURE_COOKIE_DURATION, **kwargs
        ):
        value = create_tamper_proof_string(name, value, key, duration)
        self.set_cookie(name, value, **kwargs)

    def append_to_cookie(self, name, value):
        cookie = self._response_cookies.setdefault(name, {})
        if 'value' in cookie:
            cookie['value'] = '%s:%s' % (cookie['value'], value)
        else:
            cookie['value'] = value

    def expire_cookie(self, name, **kwargs):
        if name in self._response_cookies:
            del self._response_cookies[name]
        kwargs.setdefault('path', '/')
        kwargs.update({'max_age': 0, 'expires': "Fri, 31-Dec-99 23:59:59 GMT"})
        self.set_cookie(name, '', **kwargs)

    def set_to_not_cache_response(self):
        headers = self.response_headers
        headers['Expires'] = "Fri, 31 December 1999 23:59:59 GMT"
        headers['Last-Modified'] = get_http_datetime()
        headers['Cache-Control'] = "no-cache, must-revalidate" # HTTP/1.1
        headers['Pragma'] =  "no-cache"                        # HTTP/1.0

    def cache_response(self, duration=864000):
        self.response_headers['Cache-Control'] = "public, max-age=%d;" % duration

    def compute_url(self, *args, **kwargs):
        return self.compute_url_for_host(self.site_host or self.host, *args, **kwargs)

    def compute_url_for_host(self, host, *args, **kwargs):
        out = self.scheme + '://' + host + '/' + '/'.join(
            arg.encode('utf-8') for arg in args
            )
        if kwargs:
            out += '?'
            _set = 0
            _l = ''
            for key, value in kwargs.items():
                key = urlquote(key).replace(' ', '+')
                if value is None:
                    value = ''
                if isinstance(value, list):
                    for val in value:
                        if _set: _l = '&'
                        out += '%s%s=%s' % (
                            _l, key,
                            urlquote(val.encode('utf-8')).replace(' ', '+')
                            )
                        _set = 1
                else:
                    if _set: _l = '&'
                    out += '%s%s=%s' % (
                        _l, key, urlquote(value.encode('utf-8')).replace(' ', '+')
                        )
                    _set = 1
        return out

    @property
    def current_user(self):
        if not hasattr(self, '_current_user'):
            self._current_user = self.get_current_user()
        return self._current_user

    @property
    def is_admin(self):
        if not hasattr(self, '_is_admin'):
            self._is_admin = self.get_admin_status()
        return self._is_admin

    @property
    def site_url(self):
        if not hasattr(self, '_site_url'):
            if self.site_host:
                self._site_url = self.scheme + '://' + self.site_host + '/'
            else:
                self._site_url = self.scheme + '://' + self.host + '/'
        return self._site_url

    @property
    def url(self):
        if not hasattr(self, '_url'):
            self._url = self.site_url + self.environ['PATH_INFO']
        return self._url

    @property
    def url_with_qs(self):
        if not hasattr(self, '_url_with_qs'):
            env = self.environ
            query = env['QUERY_STRING']
            self._url_with_qs = (
                self.site_url + env['PATH_INFO'] + (
                    query and '?' or '') + query
                )
        return self._url_with_qs

    @property
    def username(self):
        if not hasattr(self, '_username'):
            self._username = self.get_username()
        return self._username

    @property
    def xsrf_token(self):
        if not self._xsrf_token:
            xsrf_token = self.get_secure_cookie('xsrf')
            if not xsrf_token:
                xsrf_token = hexlify(urandom(18))
                self.set_secure_cookie('xsrf', xsrf_token)
            self._xsrf_token = xsrf_token
        return self._xsrf_token

    from login import get_admin_status, get_current_user, get_username

    try:
        from login import get_login_url
    except ImportError:
        def get_login_url(self):
            return self.compute_url('login', return_to=self.url_with_qs)

# ------------------------------------------------------------------------------
# App Runner
# ------------------------------------------------------------------------------

reqlocal = local()

def handle_http_request(
    env, start_response, dict=dict, isinstance=isinstance, urlunquote=urlunquote,
    unicode=unicode, get_response_headers=lambda: None
    ):

    reqlocal.template_error_traceback = None

    try:

        http_method = env['REQUEST_METHOD']
        ssl_mode = env['wsgi.url_scheme'] == 'https'

        if http_method == 'OPTIONS':
            start_response(*RESPONSE_OPTIONS)
            return []

        if http_method not in SUPPORTED_HTTP_METHODS:
            start_response(*RESPONSE_NOT_IMPLEMENTED)
            return []

        _path_info = env['PATH_INFO']
        if isinstance(_path_info, unicode):
            _args = [arg for arg in _path_info.split(u'/') if arg]
        else:
            _args = [
                unicode(arg, 'utf-8', 'strict')
                for arg in _path_info.split('/') if arg
                ]

        if _args:
            service_name = _args[0]
            args = _args[1:]
        else:
            service_name = '/'
            args = ()

        routed = 0
        if service_name not in SERVICE_REGISTRY:
            router = handle_http_request.router
            if router:
                _service_info = router(env, _args)
                if not _service_info:
                    logging.error("No service found for: %s" % _path_info)
                    raise NotFound
                service_name, args = _service_info
                routed = 1
            else:
                logging.error("Service not found: %s" % service_name)
                raise NotFound

        service, renderers, config = SERVICE_REGISTRY[service_name]
        kwargs = {}

        ctx = Context(service_name, env, ssl_mode)
        ctx.was_routed = routed

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

                if config['post_encoding']:
                    ctx.request_body = env['wsgi.input'].read()
                    env['wsgi.input'] = StringIO(ctx.request_body)
                    post_encoding = config['post_encoding']
                else:
                    post_encoding = 'utf-8'

                post_data = FieldStorage(
                    environ=post_environ, fp=env['wsgi.input'],
                    keep_blank_values=True
                    ).list or []

                for field in post_data:
                    key = field.name
                    if field.filename:
                        if config['blob']:
                            value = parse_blob_info(field)
                        else:
                            value = field
                    else:
                        value = unicode(field.value, post_encoding, 'strict')
                    if key in kwargs:
                        _val = kwargs[key]
                        if isinstance(_val, list):
                            _val.append(value)
                        else:
                            kwargs[key] = [_val, value]
                        continue
                    kwargs[key] = value

        def get_response_headers():
            # Figure out the HTTP headers for the response ``cookies``.
            cookie_output = SimpleCookie()
            for name, values in ctx._response_cookies.iteritems():
                name = str(name)
                cookie_output[name] = values.pop('value')
                cur = cookie_output[name]
                for key, value in values.items():
                    if key == 'max_age':
                        key = 'max-age'
                    if key not in COOKIE_KEY_NAMES:
                        continue
                    cur[key] = value
            if cookie_output:
                raw_headers = ctx._raw_headers + [
                    ('Set-Cookie', ck.split(' ', 1)[-1])
                    for ck in str(cookie_output).split('\r\n')
                    ]
            else:
                raw_headers = ctx._raw_headers
            str_headers = []; new_header = str_headers.append
            for k, v in raw_headers:
                if isinstance(k, unicode):
                    k = k.encode('utf-8')
                if isinstance(v, unicode):
                    v = v.encode('utf-8')
                new_header((k, v))
            return str_headers

        if config['skin']:
            ctx.load_skin(ctx.get_cookie('skinhost', env['HTTP_HOST']), kwargs)

        if 'submit' in kwargs:
            del kwargs['submit']

        if '__clear__' in kwargs:
            del kwargs['__clear__']

        if 'callback' in kwargs:
            ctx.json_callback = kwargs.pop('callback')

        if env.get('HTTP_X_REQUESTED_WITH') == 'XMLHttpRequest':
            ctx.ajax_request = 1
            if '__ajax__' in kwargs:
                del kwargs['__ajax__']

        if config['ssl'] and RUNNING_ON_GOOGLE_SERVERS and not ssl_mode:
            raise NotFound

        if config['xsrf']:
            if 'xsrf' not in kwargs:
                raise AuthError("XSRF token not present.")
            provided_xsrf = kwargs.pop('xsrf')
            if not secure_string_comparison(provided_xsrf, ctx.xsrf_token):
                raise AuthError("XSRF tokens do not match.")

        if config['admin'] and not ctx.is_admin:
            raise NotFound

        if (not config['anon']) and (not ctx.current_user):
            raise AuthError("You need to be logged in.")

        if config['cache']:
            cache_info = config['cache_key'](
                ctx, config['cache_spec'], service_name, *args, **kwargs
                )
            if cache_info is not None:
                cache_key, cache_namespace, cache_time = cache_info
                output = memcache.get(cache_key, cache_namespace)
                if output is not None:
                    raise HTTPContent(output)

        # Try and respond with the result of calling the service.
        if renderers and renderers[-1] == json:
            try:
                content = service(ctx, *args, **kwargs)
            except BaseHTTPError:
                raise
            except Exception, error:
                logging.critical(''.join(format_exception(*sys.exc_info())))
                response = json(
                    ctx, error=str(error),
                    error_type=error.__class__.__name__
                    )
                start_response(*RESPONSE_JSON_ERROR)
                return [response]
        else:
            content = service(ctx, *args, **kwargs)

        for renderer in renderers:
            if ctx.end_pipeline:
                break
            if content is None:
                content = {
                    'content': ''
                }
            elif not isinstance(content, dict):
                content = {
                    'content': content
                    }
            if isinstance(renderer, str):
                content = ctx.render_mako_template(renderer, **content)
            else:
                content = renderer(ctx, **content)

        if content is None:
            content = ''
        elif isinstance(content, unicode):
            content = content.encode('utf-8')

        if config['cache'] and cache_info is not None:
            memcache.set(
                cache_key, output, cache_time, namespace=cache_namespace
                )

        raise HTTPContent(content)

    # Return the content.
    except HTTPContent, payload:

        content = payload.content

        if 'Content-Type' not in ctx.response_headers:
            ctx.response_headers['Content-Type'] = 'text/html; charset=utf-8'

        ctx.response_headers['Content-Length'] = str(len(content))

        start_response(('%d %s\r\n' % ctx._status), get_response_headers())
        if http_method == 'HEAD':
            return []
        return [content]

    # Handle 404s.
    except NotFound:
        start_response(*RESPONSE_404)
        return [ERROR_404]

    # Handle 401s.
    except AuthError:
        start_response(*RESPONSE_401)
        return [ERROR_401]

    # Handle HTTP 301/302 redirects.
    except Redirect, redirect:
        headers = get_response_headers()
        if not headers:
            headers = []
        headers += [("Content-Type", "text/html; charset=utf-8")]
        headers.append(("Location", redirect.uri))
        if redirect.permanent:
            start_response(STATUS_301, headers)
        else:
            start_response(STATUS_302, headers)
        return []

    # Handle other HTTP response codes.
    except HTTPError, error:
        start_response(("%s %s" % (error.code, HTTP_STATUS_MESSAGES[error.code])), [])
        return []

    except CapabilityDisabledError:
        start_response(*RESPONSE_503)
        return [ERROR_503]

    # Log any errors and return an HTTP 500 response.
    except Exception, error:
        template_tb = reqlocal.template_error_traceback
        logging.critical(''.join(format_exception(*sys.exc_info())))
        if DEBUG:
            traceback = ''.join(html_format_exception())
        else:
            traceback = escape("%s: %s" % (error.__class__.__name__, error))
        if template_tb:
            logging.critical(PlainErrorTemplate.render(traceback=template_tb))
            if DEBUG:
                traceback = HTMLErrorTemplate.render(traceback=template_tb)
        response = ERROR_500_TRACEBACK % traceback
        start_response(*RESPONSE_500)
        if isinstance(response, unicode):
            response = response.encode('utf-8')
        return [response]

handle_http_request.router = None

# ------------------------------------------------------------------------------
# Template Error Handling
# ------------------------------------------------------------------------------

PlainErrorTemplate = MakoTemplate("""
Traceback (most recent call last):
% for (filename, lineno, function, line) in traceback.traceback:
  File "${filename}", line ${lineno}, in ${function or '?'}
    ${line | trim}
% endfor
${traceback.errorname}: ${traceback.message}
""")

HTMLErrorTemplate = MakoTemplate(r"""
<style type="text/css">
    .stacktrace { margin:5px 5px 5px 5px; }
    .highlight { padding:0px 10px 0px 10px; background-color:#9F9FDF; }
    .nonhighlight { padding:0px; background-color:#DFDFDF; }
    .sample { padding:10px; margin:10px 10px 10px 10px; font-family:monospace; }
    .sampleline { padding:0px 10px 0px 10px; }
    .sourceline { margin:5px 5px 10px 5px; font-family:monospace;}
    .location { font-size:80%; }
</style>
<%
    src = traceback.source
    line = traceback.lineno
    if src:
        lines = src.split('\n')
    else:
        lines = None
%>
<h3>${traceback.errorname}: ${traceback.message}</h3>

% if lines:
    <div class="sample">
    <div class="nonhighlight">
% for index in range(max(0, line-4),min(len(lines), line+5)):
    % if index + 1 == line:
<div class="highlight">${index + 1} ${lines[index] | h}</div>
    % else:
<div class="sampleline">${index + 1} ${lines[index] | h}</div>
    % endif
% endfor
    </div>
    </div>
% endif

<div class="stacktrace"><ul>
% for (filename, lineno, function, line) in traceback.traceback:
    <li>
    <div class="location">${filename}, line ${lineno}:</div>
    <div class="sourceline">${line | h}</div>
    </li>
% endfor
</ul></div>
""")

def template_error_handler(context, error):
    reqlocal.template_error_traceback = RichTraceback()

handle_http_request.template_error_handler = template_error_handler

# ------------------------------------------------------------------------------
# Monkey Patches
# ------------------------------------------------------------------------------

# The ``mako`` templating system uses ``beaker`` to cache segments and this
# needs various patches to make appropriate use of Memcache as a cache backend
# on App Engine.
#
# First, the App Engine memcache client needs to be setup as the ``memcache``
# module.
import google.appengine.api.memcache as memcache

sys.modules['memcache'] = memcache

import beaker.container
import beaker.ext.memcached

# And then the beaker ``Value`` object itself needs to be patched.
class Value(beaker.container.Value):

    def get_value(self):
        stored, expired, value = self._get_value()
        if not self._is_expired(stored, expired):
            return value

        if not self.createfunc:
            raise KeyError(self.key)

        v = self.createfunc()
        self.set_value(v)
        return v

beaker.container.Value = Value
beaker.ext.memcached.verify_directory = lambda x: None

# ------------------------------------------------------------------------------
# Mako
# ------------------------------------------------------------------------------

def call_template_error_handler(*args, **kwargs):
    return handle_http_request.template_error_handler(*args, **kwargs)

# The ``mako`` templating system is used. It offers a reasonably flexible engine
# with pretty decent performance.
class MakoTemplateLookup(object):

    default_template_args = {
        'format_exceptions': False,
        'error_handler': call_template_error_handler,
        'disable_unicode': False,
        'output_encoding': 'utf-8',
        'encoding_errors': 'strict',
        'input_encoding': 'utf-8',
        'module_directory': None,
        'cache_type': 'memcached',
        'cache_dir': '.',
        'cache_url': 'memcached://',
        'cache_enabled': True,
        'default_filters': ['decode.utf8'],  # will be shared across instances
        'buffer_filters': [],
        'imports': None,
        'preprocessor': None
        }

    templates_directory = 'template'

    def __init__(self, **kwargs):
        self.template_args = self.default_template_args.copy()
        self.template_args.update(kwargs)
        self._template_cache = {}
        self._template_mtime_data = {}

    if DEBUG:

        def get_template(self, uri, kwargs=None):

            filepath = join_path(self.templates_directory, uri + '.mako')
            if not exists(filepath):
                raise IOError("Cannot find template %s.mako" % uri)

            template_time = getmtime(filepath)

            if ((template_time <= self._template_mtime_data.get(uri, 0)) and
                ((uri, kwargs) in self._template_cache)):
                return self._template_cache[(uri, kwargs)]

            if kwargs:
                _template_args = self.template_args.copy()
                _template_args.update(dict(kwargs))
            else:
                _template_args = self.template_args

            template = MakoTemplate(
                uri=uri, filename=filepath, lookup=self, **_template_args
                )

            self._template_cache[(uri, kwargs)] = template
            self._template_mtime_data[uri] = template_time

            return template

    else:

        def get_template(self, uri, kwargs=None):

            if (uri, kwargs) in self._template_cache:
                return self._template_cache[(uri, kwargs)]

            filepath = join_path(self.templates_directory, uri + '.mako')
            if not exists(filepath):
                raise IOError("Cannot find template %s.mako" % uri)

            if kwargs:
                _template_args = self.template_args.copy()
                _template_args.update(dict(kwargs))
            else:
                _template_args = self.template_args

            template = MakoTemplate(
                uri=uri, filename=filepath, lookup=self, **_template_args
                )

            return self._template_cache.setdefault((uri, kwargs), template)

    def adjust_uri(self, uri, relativeto):
        return uri

def get_mako_template(ctx, uri, kwargs=None, lookup=MakoTemplateLookup().get_template):
    skin_dir = ctx.skin_dir
    if skin_dir:
        tmpl = None
        for path in [skin_dir + '/' + uri, uri]:
            try:
                tmpl = lookup(path, kwargs)
                break
            except IOError, err:
                continue
        if tmpl:
            return tmpl
        raise err
    else:
        return lookup(uri, kwargs)

def call_mako_template(ctx, template, **kwargs):
    return template.render_unicode(
        ctx=ctx, _=ctx.skin._, STATIC=ctx.STATIC, **kwargs
        )

def render_mako_template(ctx, template_name, **kwargs):
    return ctx.get_mako_template(template_name).render_unicode(
        ctx=ctx, _=ctx.skin._, STATIC=ctx.STATIC, **kwargs
        )

Context.get_mako_template = get_mako_template
Context.call_mako_template = call_mako_template
Context.render_mako_template = render_mako_template

# ------------------------------------------------------------------------------
# HTML Escape
# ------------------------------------------------------------------------------

def escape(s):
    return s.replace(u"&", u"&amp;").replace(u"<", u"&lt;").replace(
        u">", u"&gt;").replace(u'"', u"&quot;")

# ------------------------------------------------------------------------------
# WSGI App Alias
# ------------------------------------------------------------------------------

app = handle_http_request
