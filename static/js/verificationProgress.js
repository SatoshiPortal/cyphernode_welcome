top.progressStart = 0;
top.progressStartTime = 0;

var durationFormatter = function( seconds ) {


    var interval = Math.floor(seconds / 31536000);

    if (interval > 1) {
        return interval + " years";
    }
    interval = Math.floor(seconds / 2592000);
    if (interval > 1) {
        return interval + " months";
    }
    interval = Math.floor(seconds / 86400);
    if (interval > 1) {
        return interval + " days";
    }
    interval = Math.floor(seconds / 3600);
    if (interval > 1) {
        return interval + " hours";
    }
    interval = Math.floor(seconds / 60);
    if (interval > 1) {
        return interval + " minutes";
    }
    return Math.floor(seconds) + " seconds";

}

var loadVerificationProgress = function() {
    console.log( "loading verification progress" );

    var base = document.getElementById("base");
    var url = 'verificationprogress';

    if( base && base.href ) {
        url = base.href+url;
    } else {
        url = '/'+url;
    }

    var request = new XMLHttpRequest();
    request.open('GET', url );
    request.responseType = 'text';

    request.onerror = function(e) {  }

    request.onload = function() {
        if (request.readyState===4 ){
            if( request.status===200 ) {
                var result;

                try {
                    result = JSON.parse( request.response )
                } catch( e ) {
                    console.log( e );
                }

                if( result && result.verificationprogress ) {
                    result.verificationprogress = parseInt(result.verificationprogress*100000)/100000;
                    if( top.progressStart === 0 ) {
                        top.progressStart = result.verificationprogress;
                        top.progressStartTime = parseInt((new Date())/1000);
                    } else if( result.verificationprogress >= top.progressStart ) {
                        // only update display when real progress is made ;-)
                        var deltaS = parseInt((new Date())/1000) - top.progressStartTime;
                        var deltaP = result.verificationprogress - top.progressStart;

                        var eta = 0;
                        if( deltaP !== 0 ) {
                            eta = deltaS/deltaP;
                        }
                        var pBar = document.getElementById('progress-bar');
                        if( pBar ) {
                            if( eta === 0 && result.verificationprogress >= 0.99 ) {
                                pBar.classList.remove("bg-danger", "progress-bar-striped", "progress-bar-animated")
                                pBar.classList.add("bg-success");
                            } else {
                                pBar.classList.remove("bg-danger", "bg-success");
                                pBar.classList.add("progress-bar-striped", "progress-bar-animated");
                            }

                            pBar.classList.remove("bg-danger");
                            pBar.style.width = (result.verificationprogress*100)+"%";
                        }
                        var pText = document.getElementById('progress-text');
                        if( pText ) {
                            if( eta === 0 && result.verificationprogress >= 0.99 ) {
                                pText.innerText = "Initial block download complete!";
                            } else {
                                pText.innerText = "Initial block download complete in "+durationFormatter(eta);
                            }
                        }
                    }
                }
            } else {
                var pBar = document.getElementById('progress-bar');
                if( pBar ) {
                    pBar.style.width = "100%";
                    pBar.classList.remove("bg-success", "progress-bar-striped");
                    pBar.classList.add("bg-danger");
                }
                var pText = document.getElementById('progress-text');
                if( pText ) {
                    pText.innerText = "Error connecting to cyphernode";
                }
            }
        }
    };
    request.send();
}

loadVerificationProgress();
setInterval( loadVerificationProgress, 30000 );
