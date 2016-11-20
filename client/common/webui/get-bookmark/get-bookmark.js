'use strict';

(function(exports){

  function init(gmClient, contentElem, srcDir) {
    var tagsInputElem = contentElem.find('#tags_input')
    var gmTagReqInst = gmTagRequester.create({
      tagsInputElem: tagsInputElem,
      allowNewTags: false,
      gmClient: gmClient,

      loadingStatus: function(isLoading) {
        if (isLoading) {
          contentElem.find("#tmploading").html("<p>...</p>");
        } else {
          contentElem.find("#tmploading").html("<p>&nbsp</p>");
        }
      },

      onChange: function(selectedTags) {
        gmClient.getTaggedBookmarks(selectedTags.tagIDs, onBookmarksReceived);
      }
    });

    gmClient.onConnected(true, function() {
      gmClient.getTaggedBookmarks([], onBookmarksReceived);
    });

    function onBookmarksReceived(status, resp) {
      console.log('resp bkm2:', status, resp)

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
              console.log('html:', div.html());
            }
          );
        });
      } else {
        // TODO: show error
      }
    }
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmGetBookmark']={} : exports);
