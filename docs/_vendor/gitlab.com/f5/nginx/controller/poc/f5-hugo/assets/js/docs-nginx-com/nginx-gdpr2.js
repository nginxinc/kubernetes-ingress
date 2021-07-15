/**
 * @file NGINX GDPR Module.
 *
 * Obtain and save consent from EU users to use a functional cookie (mandatory)
 * and a social cookie (optional).
 * Algorithm:
 *   Read cookies, check if a user has already given consent.
 *   If no consent given and user is from EU country, show popup.
 *   If consent is given save consent values in cookies.
 *
 * @author Paul Ferlito, Ferlito/van der Wyk
 */
 /** 
  * This file is modified from the original file that is being used in nginx.com
  * Modified by oneTarek
 */


var NX_GDPR = {

  opinionGiven: false,
  clickedFormObj: null,
  formChecked: false,
  euCountries: [ "AT","BE","BG","HR","CY","CZ","DK","EE","FI","FR","DE","GR","HU","IE","IT","LV","LT","LU","MT","NL","PL","PT","RO","SK","ES","SE","GB","SI" ], //Modified by oneTarek
  //euCountriesFileUrl: '/wp-content/themes/nginx-new/js/json-data/eu-codes.json', //Commented out by oneTarek
  functionalCookieName: 'nx_functional_cookie_consent',
  socialCookieName: 'nx_social_cookie_consent',
  countryCode: "",

  init: function ($) {
    console.log('GDPR: Intializing');
    this.loadEUCountries($).done(function () {
      this.countryCode = this.getCountryCode();
      this.setGlobals();
      // set up events when document is ready
      $(function() {
        console.log('GDPR: setting up events');
        this.setupEvents($);
      }.bind(this));
      // if new visitor from EU country
      if (!this.opinionGiven && this.isEUCountry(this.countryCode)) {
        // show popup after page load.
        window.onload = function () {
          NX_GDPR.showPopup();
        }
      }
    }.bind(this));
  },

  /**
   * Load EU country codes from JSON file and cache them in session storage.
   * @returns a deferred object.
   */
  loadEUCountries: function ($) {
    return $.Deferred().resolve();
    /* //Commented out by oneTarek
    // check cache
    var euCountries;
    if (sessionStorage.euCountries) {
      euCountries = JSON.parse(sessionStorage.euCountries);
    }

    if (euCountries) {
      this.euCountries = euCountries;
      return $.Deferred().resolve();
    } else {
      return $.getJSON(this.euCountriesFileUrl)
        .done(function (data) {
          sessionStorage.euCountries = JSON.stringify(data);
          this.euCountries = data;
        }.bind(this))
        .fail(function () {
          console.error('GDPR: could not load country file from ' + this.euCountriesFileUrl);
        }.bind(this))
    }
    */
  },

  /**
   * Set up event listeners.
   */
  setupEvents: function () {
    document.getElementById('gdpr-more-info').addEventListener('click',function(e) {
      e.preventDefault();
      this.hideMain();
      this.showInfo();
      fix_fake_header_position();
    }.bind(this),false);

    document.getElementById('gdpr-less-info').addEventListener('click',function(e) {
      e.preventDefault();
      this.showMain();
      this.hideInfo();
      fix_fake_header_position();
    }.bind(this));

    document.getElementById('nx_gdpr_accept_btn').addEventListener('click',function() {
      this.userAccept();
      fix_fake_header_position();
    }.bind(this));
  },

  /**
   * Hide main popup panel.
   */
  hideMain: function () {
    this.hideElement('nx_gdpr_modal_main2');
  },

  /**
   * Show main popup panel.
   */
  showMain: function () {
    this.showElement('nx_gdpr_modal_main2');
  },

  /**
   * Show info popup panel.
   */
  showInfo: function () {
    this.showElement('nx_gdpr_modal_more_info2');
  },

  /**
   * Hide info popup panel.
   */
  hideInfo: function () {
    this.hideElement('nx_gdpr_modal_more_info2');
  },

  /**
   * Sets global variables according to consent cookie values.
   */
  setGlobals: function () {
    var fcc = NX_COOKIES.getCookie(this.functionalCookieName);
    var scc = NX_COOKIES.getCookie(this.socialCookieName);
    if (fcc !== "" || scc !== "") {
      this.opinionGiven = true;

      if (fcc === "yes") {
        NX_GDPR_FUNCTIONAL_COOKIE_CONSENT = 'yes';
      }
      if (scc === "yes") {
        NX_GDPR_SOCIAL_COOKIE_CONSENT = 'yes';
      }
    }
    else if (!this.isEUCountry(this.countryCode)) {
      NX_GDPR_FUNCTIONAL_COOKIE_CONSENT = 'yes';
      NX_GDPR_SOCIAL_COOKIE_CONSENT = 'yes';
    }
  },

  /**
   * Records individual selection. Sets cookie according to selection.
   * @param type
   * @param value
   */
  setConsent: function (type, value) { // type = 'functional' || 'social' || 'both' , value = 'yes'|| 'no'
    value = (typeof value === 'undefined') ? 'yes' : value;
    if (value !== 'yes' && value !== 'no') {
      return;
    }

    switch (type) {
      case 'both':
        NX_COOKIES.setCookie(this.functionalCookieName, value, 129600); //180 month
        NX_COOKIES.setCookie(this.socialCookieName, value, 129600); //180 month
        this.updateFormSocialConsentField(value);
        this.updateFormFunctionalConsentField(value);
        break;
      case 'social':
        NX_COOKIES.setCookie(this.socialCookieName, value, 129600); //180 month
        this.updateFormSocialConsentField(value);
        break;
      case 'functional':
        NX_COOKIES.setCookie(this.functionalCookieName, value, 129600); //180 month
        this.updateFormFunctionalConsentField(value);
        break;
    }
    this.setGlobals();
  },

  deleteConsent: function (type) { // t = 'functional' || 'social' || 'both'
    switch (type) {
      case 'both':
        NX_COOKIES.deleteCookie(this.functionalCookieName);
        NX_COOKIES.deleteCookie(this.socialCookieName);
        break;
      case 'functional':
        NX_COOKIES.deleteCookie(this.functionalCookieName);
        break;
      case 'social':
        NX_COOKIES.deleteCookie(this.socialCookieName);
        break;
    }
  },

  /**
   * Acceptance handler. Records user selections.
   */
  userAccept: function () {
    if (document.getElementById('nx_gdpr_social_con_checkbox').checked === true) {
      this.setConsent('both', 'yes');
    }
    else {
      this.setConsent('social', 'no');
      this.setConsent('functional', 'yes');
    }

    // If the GDPR popup was triggered by a form submission submit the form
    // and hide the popup.
    if (this.clickedFormObj !== null) {
      this.setGlobals();
      var _form = this.clickedFormObj;
      this.formChecked = true;
      var submit_button = _form.find(".mktf_submit_button").first();
      submit_button.trigger("click");
      this.hidePopup();
    }
    else {
      document.getElementById('nx_gdpr_accept_btn').disabled = true;
      location.reload();
    }
  },

  /**
   * Update the form social cookie opt-in checkbox according to previous acceptance.
   * @param value
   */
  updateFormSocialConsentField: function (value) {
    if (this.clickedFormObj !== null) {
      var _form = this.clickedFormObj;
      _form.find(".mktf_Cookie_Opt_In__c").first().val(value);
    }
  },

  /**
   * Update the form functional cookie opt-in checkbox according to previous acceptance.
   * @param value
   */
  updateFormFunctionalConsentField: function (value) {
    if (this.clickedFormObj !== null) {
      var _form = this.clickedFormObj;
      _form.find('.mktf_Functional_Cookie_Opt_In__c').first().val(value);
    }
  },

  /**
   * Verify GDPR acceptance before form submission.
   * Called when a form with a Country Code field is submitted.
   * @param countryCode
   * @param _form form element
   * @returns {boolean} true to allow form submission
   */
  do_on_form_country_selected: function (countryCode, _form) {
    _form = (typeof countryCode === 'undefined') ? null : _form;
    if (this.formChecked === true || countryCode === "") {
      return true;  // allow form submission
    }
    // if new visitor from the EU show the popup
    if (this.isEUCountry(countryCode) && this.opinionGiven === false) {
      this.clickedFormObj = _form;
      this.showPopup();
      return false; // block form submission
    }
    else {
      return true;
    }
  },

  /**
   * Show GDPR popup.
   */
  showPopup: function () {
    // set the state of the checkboxes according to the cookies.
    var scc = NX_COOKIES.getCookie(this.socialCookieName);
    if (scc === 'no') {
      document.getElementById('nx_gdpr_social_con_checkbox').checked = false;
    }
    this.showElement('nx_gdpr_modal2');
    fix_fake_header_position();
  },

  /**
   * Hide GDPR popup.
   */
  hidePopup: function () {
    this.hideElement('nx_gdpr_modal2');
    fix_fake_header_position();
  },

  /**
   * Show an element.
   * @param id
   */
  showElement: function (id) {
    var elm = document.getElementById(id);
    elm.classList.remove('hidden');
  },

  /**
   * Hide an element.
   * @param id
   */
  hideElement: function (id) {
    var elm = document.getElementById(id);
    elm.classList.add('hidden');
  },

  /**
   * Return true if a country code is in the EU.
   * @param countryCode
   * @returns {boolean}
   */
  isEUCountry: function (countryCode) {

    // use last known country code
    if (!countryCode) {
      countryCode = this.countryCode;
    }
    // check visitor comes from european union country.

    if (!countryCode) return false;

    return this.euCountries.indexOf(countryCode) > -1;

  },//end method isEUCountry

  /**
   * Get the country code from the geoip cookie.
   * @returns {string}
   */
  getCountryCode: function () {//return 'AT';

    var countryCode = "";
    var geoLocation = NX_COOKIES.getCookie('geoip');

    if (geoLocation !== "") {
      var geoCodes = geoLocation.split(',');
      countryCode = geoCodes[0];
    }
    return countryCode;

  }

};//end of NX_GDPR

// initialize NX_GDPR object
console.log('GDPR: running');
NX_GDPR.init(jQuery);


function nx_sould_use_functional_cookie() {
  return (NX_GDPR_FUNCTIONAL_COOKIE_CONSENT == 'yes') ? true : false;
}

function nx_sould_use_social_cookie() {
  return (NX_GDPR_SOCIAL_COOKIE_CONSENT == 'yes') ? true : false;
}

/* 
Following lines were deleted by oneTarek
We don't need those lines for docs
*/

//NEW LINES ADDED BY oneTarek

function fix_fake_header_position(){
  _w = jQuery(window).width();
  var has_gdpr = !jQuery("#nx_gdpr_modal2").hasClass('hidden');
  if( _w < 576 && has_gdpr )
  {
    var gh = jQuery("#nx_gdpr_modal2").height();
    var fake_padding = gh+99;
    jQuery('#nx_fake_header').attr('style', 'padding-top:'+fake_padding+'px;' );
  }
  else
  {
    jQuery('#nx_fake_header').attr('style', '' );
  }
    
}

jQuery(window).resize(function(){
    fix_fake_header_position();
});

jQuery(document).ready(function(){
    fix_fake_header_position();
});


