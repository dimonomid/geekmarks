'use strict';

(function(exports){

  var gmClientLoggedIn = undefined;

  function init(_gmClient, contentElem, srcDir) {
    _gmClient.createGMClientLoggedIn().then(function(instance) {
      gmClientLoggedIn = instance;
      var tagsInputElem = contentElem.find('#tags_input')
      var gmTagReqInst = gmTagRequester.create({
        tagsInputElem: tagsInputElem,
        allowNewTags: false,
        gmClientLoggedIn: gmClientLoggedIn,

        loadingStatus: function(isLoading) {
          if (isLoading) {
            contentElem.find("#tmploading").html("<p>...</p>");
          } else {
            contentElem.find("#tmploading").html("<p>&nbsp</p>");
          }
        },

        onChange: function(selectedTags) {
          gmClientLoggedIn.getTaggedBookmarks(selectedTags.tagIDs, onBookmarksReceived);
        }
      });

      gmClientLoggedIn.onConnected(true, function() {
        gmClientLoggedIn.getTaggedBookmarks([], onBookmarksReceived);
      });

      function onBookmarksReceived(status, resp) {
        if (status === 200) {
          var listElem = contentElem.find("#tmpdata");
          listElem.text("");

          resp.forEach(function(bkm) {
            var div = jQuery('<div/>', {
              id: 'bookmark_' + bkm.id,
              class: 'bookmark-div',
            });
            div.load(
              srcDir + "/bkm.html",
              undefined,
              function() {
                div.find("#bkm_link").html(bkm.title);
                div.find("#bkm_link").attr('href', bkm.url);
                div.find("#bkm_link").attr('target', '_blank');

                // Just after user clicked at some bookmark, close the
                // bookmark selection window
                div.find("#bkm_link").click(function() {
                  window.close();
                  return true;
                })

                div.find("#edit_link").click(function() {
                  gmPageWrapper.openPageEditBookmarks(bkm.id);
                  return false;
                })

                div.appendTo(listElem);
              }
            );
          });
        } else {
          // TODO: show error
        }
      }
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmGetBookmark']={} : exports);
