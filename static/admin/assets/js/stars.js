window.onload = function() {
  var starfieldTimer;

  window.addEventListener('resize', resizeCanvas, false);

  function resizeCanvas() {
    var starsId     = "stars"; 
    var canvas = document.getElementById(starsId);
    canvas.width = window.innerWidth;
    // TODO: jared
    // remove this comment to render
    // renderScene(); 
  }

  resizeCanvas();

  function renderScene() {
    window.clearInterval(starfieldTimer);
    var starsId     = "stars";
    var framerate = 60;
    var flightSpeed = 0.0015;
    var width = window.innerWidth;
    var numberOfStarsModifier = 1.0;
    var starMinSize;
    var starMaxSize;

    if (width <= 700) {
      flightSpeed = 0.0035;
      numberOfStarsModifier = 5;
      starMinSize = 0.8;
      starMaxSize = 2.0;
    } else {
      flightSpeed = 0.0010;
      numberOfStarsModifier = 4;
      starMinSize = 0.6;
      starMaxSize = 2.0;
    }

    var canvas        = document.getElementById(starsId),
    context       = canvas.getContext("2d"),
    height        = canvas.height,
    numberOfStars = width * height / 1000 * numberOfStarsModifier,
    dirX          = width / 2,
    dirY          = height / 2,
    stars         = [],
    TWO_PI        = Math.PI * 2;

    //context.clearRect(0, 0, canvas.width, canvas.height);

    // initialize starfield
    for(var x = 0; x < numberOfStars; x++) {
      stars[x] = {
        x: range(0, width),
        y: range(0, height),
        size: range(starMinSize, starMaxSize)
      };
    }

    starfieldTimer = window.setInterval(tick, Math.floor(1000 / framerate));

    function tick() {
      var oldX,
      oldY;

      context.clearRect(0, 0, width, height);

      for(var x = 0; x < numberOfStars; x++) {
        oldX = stars[x].x;
        oldY = stars[x].y;

        stars[x].x += (stars[x].x - dirX) * stars[x].size * flightSpeed;
        stars[x].y += (stars[x].y - dirY) * stars[x].size * flightSpeed;
        stars[x].size += flightSpeed;

        if(stars[x].x < 0 || stars[x].x > width || stars[x].y < 0 || stars[x].y > height) {
          stars[x] = {
            x: range(0, width),
            y: range(0, height),
            size: 0
          };
        }

        context.strokeStyle = "rgba(255, 255, 255, " + Math.min(stars[x].size, 1) + ")";
        context.lineWidth = stars[x].size;
        context.beginPath();
        context.moveTo(oldX, oldY);
        context.lineTo(stars[x].x, stars[x].y);
        context.stroke();
      }
    }

    function range(start, end) {
      return Math.random() * (end - start) + start;
    }
  }
};
