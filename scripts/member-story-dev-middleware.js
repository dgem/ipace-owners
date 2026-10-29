'use strict';

module.exports = function memberStoryDevMiddleware(req, res, next) {
  var requestUrl = req.url || '';
  var queryIndex = requestUrl.indexOf('?');
  var pathname = queryIndex === -1 ? requestUrl : requestUrl.slice(0, queryIndex);

  if (/^\/member-stories\/[^/]+\/?$/.test(pathname)) {
    req.url = '/member-stories/' + (queryIndex === -1 ? '' : requestUrl.slice(queryIndex));
  }

  next();
};
