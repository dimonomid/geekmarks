'use strict';

(function(exports){

  var STORAGE_KEY = 'gmclient_data';

  exports.create = function(){
    return gmClient.create({
      server: "localhost:4000",
      serverSSL: false,

      setLocalData: setLocalData,
      getLocalData: getLocalData,
      launchWebAuthFlow: launchWebAuthFlow,
    });
  };

  function setLocalData(data) {
    return new Promise(function(resolve, reject) {
      var storageData = {};
      storageData[STORAGE_KEY] = data;
      chrome.storage.sync.set(storageData, function() {
        resolve();
      });
    });
  }

  function getLocalData() {
    return new Promise(function(resolve, reject) {
      chrome.storage.sync.get(STORAGE_KEY, function(v) {
        resolve(v[STORAGE_KEY]);
      });
    });
  }

  function launchWebAuthFlow(provider) {
    var self = this;
    return new Promise(function(resolve, reject) {
      switch (provider) {
        case "google":
          self.getOAuthClientID(provider).then(function(resp) {

            var url = URI("https://accounts.google.com/o/oauth2/auth")
              .addSearch("scope", "email")
              .addSearch("redirect_uri", chrome.identity.getRedirectURL())
              .addSearch("client_id", resp.clientID)
              .addSearch("response_type", "code")
              .toString();

            chrome.identity.launchWebAuthFlow(
              {
                url: url,
                interactive: true
              },
              function(responseURL) {
                // We've got code to be exchanged for the access token: now,
                // pass the code to the server, it will perform the exchange,
                // and return the geekmark's token (not the Google's one)
                var uri = new URI(responseURL);
                var queryParams = uri.search(true);

                self.authenticate(
                  provider,
                  chrome.identity.getRedirectURL(),
                  queryParams.code
                ).then(function(resp) {
                  // Got a token
                  resolve(self.onAuthenticated(resp));
                }).catch(function(e) {
                  reject(e);
                });
              }
            );
          }).catch(function(e) {
            reject(e);
          });
          break;
        default:
          reject("unknown auth provider: " + provider);
          break;
      }
    });
  }


})(typeof exports === 'undefined' ? this['gmClientFactory']={} : exports);
