'use strict';

(function(exports){

  var contentElem = undefined;

  function init(gmClient, _contentElem, srcDir, queryParams, curTabData) {
    contentElem = _contentElem;
    var treeTable = contentElem.find('#tree_table')

    treeTable.treetable({
      expandable: true,
    });
  }

  exports.init = init;

})(typeof exports === 'undefined' ? this['gmTagsTree']={} : exports);
