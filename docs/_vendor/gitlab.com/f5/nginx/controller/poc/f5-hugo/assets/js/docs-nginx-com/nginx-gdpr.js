/**
Nginx GDPR manager
@author oneTarek
**/

//some global vars. DO NOT change these variables names. These variables are being used in GTM.
var NX_GDPR_FUNCTIONAL_COOKIE_CONSENT = "no"; // possible values are 'yes' and 'no'
var NX_GDPR_SOCIAL_COOKIE_CONSENT = "no"; // possible values are 'yes' and 'no'
//end global vars

var NX_GDPR;
NX_GDPR = {

	functional_cookie_consent : false,
	social_cookie_consent : false,
	opinionGiven : false ,
	clickedFormObj : null,
	formChecked : false,

	init : function(){
		this.checkConsent();
		if( this.opinionGiven == true )
		{
			//nothing to do
		}
		else
		{
			if( this.is_eu_country() )
			{
				//show popup.
				window.onload = function(){
					NX_GDPR.showPopup(); //run this after page is loaded
				}
			}
		}

	},

	checkConsent : function(){
		var fcc = this.getCookie('nx_functional_cookie_consent');
		var scc = this.getCookie('nx_social_cookie_consent');
		if( fcc != "" || scc !="" )
		{
			this.opinionGiven = true;
			
			if( fcc == "yes" )
			{
				this.functional_cookie_consent = true;
				NX_GDPR_FUNCTIONAL_COOKIE_CONSENT = 'yes';
			}
			if( scc == "yes" )
			{
				this.social_cookie_consent = true;
				NX_GDPR_SOCIAL_COOKIE_CONSENT = 'yes';
			}


		}
		else if( ! this.is_eu_country() )
		{
			NX_GDPR_FUNCTIONAL_COOKIE_CONSENT = 'yes';
			NX_GDPR_SOCIAL_COOKIE_CONSENT = 'yes';
		}


	},
	
	setConsent : function( t, d ){ // t = 'functional' || 'social' || 'both' , d = 'yes'|| 'no'
		d = (typeof d === 'undefined') ? 'yes' : d;
		if( d != 'yes' && d != 'no' ){ return ; }
		if( t =='both' )
		{
			this.setCookie('nx_functional_cookie_consent', d, 129600 ); //180 month
			this.setCookie('nx_social_cookie_consent', d, 129600 ); //180 month
			this.updateFormSocialConsentField( d );
		}
		else if( t == 'functional')
		{
			this.setCookie('nx_functional_cookie_consent', d, 129600 ); //180 month
		}
		else if( t == 'social' )
		{
			this.setCookie('nx_social_cookie_consent', d, 129600 ); //180 month
			this.updateFormSocialConsentField( d );
		}
	},

	deleteConsent : function( t ){ // t = 'functional' || 'social' || 'both'
		if( t =='both' )
		{
			this.deleteCookie('nx_functional_cookie_consent');
			this.deleteCookie('nx_social_cookie_consent');
		}
		else if( t == 'functional ')
		{
			this.deleteCookie('nx_functional_cookie_consent');
		}
		else if( t == 'social' )
		{
			this.deleteCookie('nx_social_cookie_consent');
		}
	},

	userAccept : function(){
		if( document.getElementById('nx_gdpr_social_con_checkbox').checked == true )
		{
			this.setConsent('both', 'yes');
		}
		else
		{
			this.setConsent('social', 'no');
			this.setConsent('functional', 'yes');
		}
		//this.hidePopup();
		
		if( this.clickedFormObj !== null )
		{
			var _form = this.clickedFormObj;
			this.formChecked = true;
			var submit_button = _form.find(".mktf_submit_button").first();
			submit_button.trigger( "click" );
			this.hidePopup();
			
		}
		else
		{
			document.getElementById('nx_gdpr_accept_btn').disabled = true;
			location.reload();
		}
		
	},

	updateFormSocialConsentField : function( value )
	{
		if( this.clickedFormObj !== null )
		{
			var _form = this.clickedFormObj;
			_form.find(".mktf_Cookie_Opt_In__c").first().val( value );
		}
	},

	do_on_form_country_selected : function( countryCode , _form ){
		_form = (typeof countryCode === 'undefined') ? null : _form;
		if( this.formChecked == true )
		{
			return true;
		}
		if( countryCode == ""){ return true; } // true to allow the form summision
		if( this.is_eu_country( countryCode ) && this.opinionGiven == false )
		{
			this.clickedFormObj = _form;
			this.showPopup();
			return false;
		}
		else
		{
			return true;
		}
	},

	showPopup : function(){

		var scc = this.getCookie('nx_social_cookie_consent');
		if( scc == 'no' )
		{
			document.getElementById('nx_gdpr_social_con_checkbox').checked = false;
		}
		this.showElement('nx_gdpr_modal');
	},

	hidePopup : function(){
		this.hideElement('nx_gdpr_modal');
	},
	
	showElement : function( id ){
		var elm = document.getElementById(id);
		this.removeClass( elm, "hidden" );
	},
	hideElement : function ( id ){
		var elm = document.getElementById(id);
		this.addClass( elm, "hidden" );
	},

	hasClass : function(el, className) { //https://jaketrent.com/post/addremove-classes-raw-javascript/
	  if (el.classList)
	    return el.classList.contains(className)
	  else
	    return !!el.className.match(new RegExp('(\\s|^)' + className + '(\\s|$)'))
	},

	addClass : function(el, className) {
	  if (el.classList)
	    el.classList.add(className)
	  else if (!this.hasClass(el, className)) el.className += " " + className
	},

	removeClass : function(el, className) {
	  if (el.classList)
	    el.classList.remove(className)
	  else if (this.hasClass(el, className)) {
	    var reg = new RegExp('(\\s|^)' + className + '(\\s|$)')
	    el.className=el.className.replace(reg, ' ')
	  }
	},

	setCookie : function(name, value, hours){
	  var expire = "";
	  if(hours != null)
	  {
	    expire = new Date((new Date()).getTime() + hours * 3600000);
	    expire = "; expires=" + expire.toGMTString();
		path =  "; path=/"; 
	  }
	  document.cookie = name + "=" + escape(value) + expire + path;
	},
	
	getCookie : function(name){
	  var cookieValue = "";
	  var search = name + "=";
	  if(document.cookie.length > 0)
	  { 
	    offset = document.cookie.indexOf(search);
	    if (offset != -1)
	    { 
	      offset += search.length;
	      end = document.cookie.indexOf(";", offset);
	      if (end == -1) end = document.cookie.length;
	      cookieValue = unescape(document.cookie.substring(offset, end))
	    }
	  }
	  return cookieValue;
	},

	deleteCookie : function( name ){
		document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:01 GMT;';	
	},

	is_eu_country : function( countryCode ){ //optional perameter countryCode //check visitor comes from european union country. 
    
	    countryCode = (typeof countryCode === 'undefined') ? '' : countryCode;
//return true;
	    var eu_country_list = [
	        "AT", //Austria
	        "BE", //Belgium
	        "BG", //Bulgaria
	        "HR", //Croatia
	        "CY", //Cyprus
	        "CZ", //Czech Republic
	        "DK", //Denmark
	        "EE", //Estonia
	        "FI", //Finland
	        "FR", //France
	        "DE", //Germany
	        "GR", //Greece
	        "HU", //Hungary
	        "IE", //Ireland
	        "IT", //Italy
	        "LV", //Latvia
	        "LT", //Lithuania
	        "LU", //Luxembourg
	        "MT", //Malta
	        "NL", //Netherlands
	        "PL", //Poland
	        "PT", //Portugal
	        "RO", //Romania
	        "SK", //Slovakia
	        "SI", //Slovenia
	        "ES", //Spain
	        "SE", //Sweden
	        "GB", //United Kingdom
	    ];//end of array

	    if( countryCode == "")//read country code from geoip cookie
	    {
	        var geoLocation = NX_GDPR.getCookie('geoip');

	        if( geoLocation !="" )
	        {
	            var geoCodes = geoLocation.split(',');
	            var countryCode = geoCodes[0];            

	        }
	        
	    }

	    if( countryCode != "")
	    {
	        if( -1 != eu_country_list.indexOf( countryCode ) )
	        {
	            return true;
	        }
	        else
	        {
	            return false;
	        }

	    }
	    else
	    {
	        return false;
	    }


	},//end method is_eu_country

	get_country_code : function(){
    
	    var countryCode = "";
	    var geoLocation = NX_GDPR.getCookie('geoip');

	    if( geoLocation !="" )
	    {
	        var geoCodes = geoLocation.split(',');
	        countryCode = geoCodes[0];            

	    }
	    return countryCode;

	}

};//end of NX_GDPR

NX_GDPR.init(); //run NX_GDPR object

function nx_gdpr_show_popup(){
	NX_GDPR.showPopup();
}

function nx_gdpr_accept(){
	NX_GDPR.userAccept();
}

function nx_gdpr_show_more_info(){
	NX_GDPR.hideElement("nx_gdpr_modal_main");
	NX_GDPR.showElement("nx_gdpr_modal_more_info");
}

function nx_gdpr_hide_more_info(){
	NX_GDPR.hideElement("nx_gdpr_modal_more_info");
	NX_GDPR.showElement("nx_gdpr_modal_main");
}

function nx_sould_use_functional_cookie(){
	return ( NX_GDPR_FUNCTIONAL_COOKIE_CONSENT == 'yes' ) ? true : false;
}

function nx_sould_use_social_cookie(){
	return ( NX_GDPR_SOCIAL_COOKIE_CONSENT == 'yes' ) ? true : false;
}
