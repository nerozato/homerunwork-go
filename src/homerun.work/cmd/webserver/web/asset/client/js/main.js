$(".slider-content").slick({
  slidesToShow: 1,
  slidesToScroll: 1,
  arrows: !0,
  fade: !1,
  asNavFor: ".slider-navigation",
}),
  $(".slider-navigation").slick({
    slidesToShow: 4,
    slidesToScroll: 1,
    asNavFor: ".slider-content",
    dots: !1,
    focusOnSelect: !0,
  }),
  $("#datepicker").datepicker({
    maxViewMode: 1,
  }),
  $("#datepicker").on("changeDate", function () {
    $("#date-selected").val($("#datepicker").datepicker("getFormattedDate"));
    $(".day").removeClass("today");
  }),
  $(function () {
    $("textarea").keyup(function () {
      var e = $(this).val().length;
      $(this).parent("div").find("span").text(e);
    });
  }),
  $("article .lead").html(function (e, a) {
    return a.replace(
      /^[^a-zA-Z]*([a-zA-Z])/g,
      '<span class="big-cap">$1</span>'
    );
  });
function getTimeZone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone;
}
function setCookie(name, value) {
  document.cookie = name + "=" + value + ";path=/;";
}
function setCookieTimeZone(cookieName) {
  setCookie(cookieName, getTimeZone());
}
function createSvcTimeSelector(
  timeDivId,
  paginationDivId,
  timeInputName,
  times,
  timeSelectFn
) {
  var currentPage = 0;
  var limit = 16;
  var totalPage = 0;
  $(function () {
    if (times.length % limit === 0) {
      totalPage = parseInt(times.length / limit);
    } else {
      totalPage = parseInt(times.length / limit) + 1;
    }
    // Get Page that contain selected item.
    for (var i = 0; i < times.length; i++) {
      if (times[i].selected) {
        currentPage = parseInt(i / limit);
        break;
      }
    }
    renderTimeTable(currentPage);
    renderPagination(currentPage);
  });
  function renderTimeTable(page) {
    $(timeDivId).empty();
    content = "";
    var end = (page + 1) * limit;
    if (end > times.length) {
      end = times.length;
    }
    for (var i = page * limit; i < end; i++) {
      const item = times[i];
      content += renderTimeSlot(item, i);
    }
    $(timeDivId).append(content);
  }
  function renderPagination(page) {
    if (times.length > limit) {
      var prevDisabled = "";
      var nextDisabled = "";
      if (page === 0) {
        prevDisabled = "disabled";
      } else if (page === totalPage - 1) {
        nextDisabled = "disabled";
      }
      $(paginationDivId).append(`
          <button type="button" id="pagePrev" class="btn" ${prevDisabled}>
              <i class="btn-quaternary fa fa-caret-left" aria-hidden="true"></i>
              <span>Earlier</span>
          </button>
          <button type="button" id="pageNext" class="btn" ${nextDisabled}>
              <span>Later</span>               
              <i class="btn-quaternary fa fa-caret-right" aria-hidden="true"></i>
          </button>
      `);
    }
  }
  function updatePagination(page) {
    if (times.length > limit) {
      if (page === 0) {
        $("#pagePrev").attr("disabled", true);
        $("#pageNext").attr("disabled", false);
      } else if (page === totalPage - 1) {
        $("#pagePrev").attr("disabled", false);
        $("#pageNext").attr("disabled", true);
      } else {
        $("#pagePrev").attr("disabled", false);
        $("#pageNext").attr("disabled", false);
      }
    }
  }
  function renderTimeSlot(item, index) {
    var disabled = "";
    var checked = "";
    if (item.disabled) {
      disabled = "disabled";
    }
    if (item.selected) {
      checked = "checked";
    }
    return `
      <div class="form-check form-check-inline ${disabled}">
          <label>
              <input type="radio" class="form-check-input" name="${timeInputName}" value="${item.value}" ${disabled} ${checked} data_id="${index}"><span>${item.label}</span>
          </label>
      </div>
    `;
  }
  $(paginationDivId).on("click", "#pagePrev", function (e) {
    currentPage--;
    updatePagination(currentPage);
    renderTimeTable(currentPage);
  });
  $(paginationDivId).on("click", "#pageNext", function (e) {
    currentPage++;
    updatePagination(currentPage);
    renderTimeTable(currentPage);
  });
  $(timeDivId).on("change", ".form-check-input", function (e) {
    const index = parseInt($(this).attr("data_id"));
    for (var i = 0; i < times.length; i++) {
      if (i === index) {
        times[i].selected = true;
      } else {
        times[i].selected = false;
      }
    }
    timeSelectFn();
  });
}
