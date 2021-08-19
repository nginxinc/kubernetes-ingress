"use strict";
/**
 * @file NGINX Cookie Handling Module.
 *
 * Get and set cookies for the site.
 *
 * @author Paul Ferlito, Ferlito/van der Wyk
 */

var NX_COOKIES = {
  /**
   * Set a cookie value.
   * @param name
   * @param value
   * @param hours
   */
  setCookie: function (name, value, hours) {
    var expire = "";
    if (hours != null) {
      expire = new Date((new Date()).getTime() + hours * 3600000);
      expire = "; expires=" + expire.toGMTString();
      var path = "; path=/";
    }
    document.cookie = name + "=" + escape(value) + expire + path;
  },

  /**
   * Get a cookie value.
   * @param name
   * @returns {string}
   */
  getCookie: function (name) {
    var cookieValue = "";
    var search = name + "=";
    if (document.cookie.length > 0) {
      var offset = document.cookie.indexOf(search);
      if (offset !== -1) {
        offset += search.length;
        var end = document.cookie.indexOf(";", offset);
        if (end === -1) end = document.cookie.length;
        cookieValue = unescape(document.cookie.substring(offset, end))
      }
    }
    return cookieValue;
  },

  /**
   * Delete a cookie.
   * @param name
   */
  deleteCookie: function (name) {
    document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:01 GMT;';
  }

};