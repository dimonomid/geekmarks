'use strict';

(function(exports){

  var contentElem = undefined;
  var gmClientInst = undefined;
  var gmClientLoggedIn = undefined;

  var rootTagKey = undefined;
  var keyToTag = {};

  function init(_gmClient, _contentElem, srcDir, queryParams, curTabData) {
    gmClientInst = _gmClient;
    contentElem = _contentElem;

    applyUI();

    contentElem.find('#login_google_link').click(function() {
      gmClientInst.login("google").then(function(instance) {
        alert('Logged in successfully.');
        applyUI();
      }).catch(function(e) {
        console.log('login error:', e)
        alert('error:' + JSON.stringify(e));
      });
    });

    contentElem.find('#logout_link').click(function() {
      gmClientInst.logout().then(function() {
        //alert('logged out');
        applyUI();
      }).catch(function(e) {
        console.log('logout error:', e)
        alert('error:' + JSON.stringify(e));
      });
    });
  }

  // Show the login/logout box depending on whether the user is logged in now
  function applyUI() {
    gmClientInst.createGMClientLoggedIn().then(function(instance) {
      gmClientLoggedIn = instance;
      if (gmClientLoggedIn == null) {
        contentElem.find('#logged_out_div').removeClass('hidden');
        contentElem.find('#logged_in_div').addClass('hidden');
      } else {
        contentElem.find('#logged_out_div').addClass('hidden');
        contentElem.find('#logged_in_div').removeClass('hidden');
      }
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmLoginLogout']={} : exports);
