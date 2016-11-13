'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams) {
    contentElem = _contentElem;
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
    });

    gmClient.onConnected(true, function() {
      gmClient.getBookmarkByID(queryParams.bkm_id, function(resp) {
        console.log('getBookmarkByID resp:', resp);

        if (resp.url) {
          contentElem.find("#bkm_url").val(resp.url);
        }

        if (resp.title) {
          contentElem.find("#bkm_title").val(resp.title);
        }

        if (resp.comment) {
          contentElem.find("#bkm_comment").val(resp.comment);
        }

        resp.tags.forEach(function(tag) {
          gmTagReqInst.addTag(tag.id, tag.fullName, false)
        });
      });
    });

  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmEditBookmark']={} : exports);
