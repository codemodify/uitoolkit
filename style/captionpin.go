package style

// The keep-above pin.
//
// Not a shape drawn here: a pushpin icon, traced and kept. Four
// hand-drawn attempts at this glyph — a ceiling with a window under it, a
// tack of bars and needles, a pin swept from a profile along an axis, a
// plain head and needle — every one read as a smudge at the size a
// caption button is. Eight to eighteen pixels is not enough to
// *construct* a pin in; it is enough to *show* one.
//
// How it was made: the icon thresholded to a mask, its four pieces
// (handle, body, flange, needle) taken as separate shapes, and each piece
// reduced to its convex hull. Every piece fills at least 97.8% of its own
// hull, so the hulls are the shapes rather than an approximation, and a
// hull is a handful of points where a traced curve was hundreds.
//
// The needle is then widened across its axis by a sixth and lengthened
// along it by a seventh, away from the flange so that the end it hangs
// from does not move. Neither could come from the mask: dilating the
// needle there merged it into the flange, and a merged piece has a
// concave hull that no longer describes it. Reshaping the outline cannot
// merge anything, because the pieces are filled in one ink and are free
// to overlap.
//
// The coordinates are the unit square, y down, so the pin scales by
// multiplication and nothing here is tied to a pixel grid. What puts it
// on a grid is drawCaptionPin, which fills it a scan line at a time.

// captionPinLoose is the pin lying as the artwork draws it: not kept above.
var captionPinLoose = [][]float32{
	{
		0.3611, 0.4005, 0.3687, 0.3929, 0.3915, 0.3752, 0.6121, 0.2079,
		0.6831, 0.1546, 0.6907, 0.1496, 0.7008, 0.1496, 0.8478, 0.2966,
		0.8504, 0.3017, 0.8504, 0.3042, 0.8478, 0.3093, 0.7946, 0.3802,
		0.6907, 0.5171, 0.6019, 0.6337, 0.5969, 0.6363, 0.5918, 0.6363,
		0.3637, 0.4081, 0.3611, 0.4031,
	},
	{
		0.1026, 0.4385, 0.1051, 0.4335, 0.1178, 0.4233, 0.1406, 0.4081,
		0.1507, 0.4031, 0.1634, 0.3980, 0.1786, 0.3929, 0.1887, 0.3904,
		0.2040, 0.3879, 0.2521, 0.3879, 0.2673, 0.3904, 0.2775, 0.3929,
		0.2952, 0.3980, 0.3206, 0.4107, 0.3358, 0.4208, 0.3459, 0.4284,
		0.5665, 0.6490, 0.5766, 0.6616, 0.5867, 0.6768, 0.5969, 0.6971,
		0.6019, 0.7098, 0.6045, 0.7174, 0.6070, 0.7275, 0.6095, 0.7427,
		0.6121, 0.7681, 0.6121, 0.7732, 0.6095, 0.7985, 0.6070, 0.8112,
		0.6045, 0.8213, 0.6019, 0.8289, 0.5969, 0.8416, 0.5893, 0.8594,
		0.5842, 0.8670, 0.5690, 0.8872, 0.5614, 0.8948, 0.5538, 0.8948,
		0.1026, 0.4436,
	},
	{
		0.6679, 0.0406, 0.6729, 0.0253, 0.6907, 0.0076, 0.7008, 0.0025,
		0.7160, 0.0000, 0.7211, 0.0000, 0.7363, 0.0025, 0.7464, 0.0076,
		0.7540, 0.0127, 0.9873, 0.2459, 0.9923, 0.2535, 0.9949, 0.2611,
		0.9974, 0.2712, 0.9974, 0.2890, 0.9923, 0.3042, 0.9873, 0.3118,
		0.9771, 0.3219, 0.9670, 0.3270, 0.9568, 0.3295, 0.9442, 0.3321,
		0.9416, 0.3321, 0.9214, 0.3270, 0.9137, 0.3219, 0.6755, 0.0837,
		0.6679, 0.0684,
	},
	{
		0.0026, 0.9384, 0.0082, 0.9093, 0.0110, 0.9006, 0.0139, 0.8948,
		0.2539, 0.6551, 0.2597, 0.6522, 0.3064, 0.6526, 0.3476, 0.6937,
		0.3480, 0.7433, 0.1022, 0.9888, 0.0935, 0.9916, 0.0818, 0.9944,
		0.0527, 1.0000, 0.0060, 0.9996, 0.0031, 0.9967,
	},
}

// captionPinDriven is the same pin turned upright — driven in, kept above.
var captionPinDriven = [][]float32{
	{
		0.3641, 0.4500, 0.3641, 0.4413, 0.3670, 0.4179, 0.3977, 0.1943,
		0.4079, 0.1227, 0.4094, 0.1154, 0.4153, 0.1096, 0.5847, 0.1096,
		0.5891, 0.1110, 0.5906, 0.1125, 0.5921, 0.1169, 0.6023, 0.1885,
		0.6213, 0.3273, 0.6373, 0.4456, 0.6359, 0.4500, 0.6330, 0.4530,
		0.3700, 0.4530, 0.3656, 0.4515,
	},
	{
		0.2370, 0.6210, 0.2355, 0.6166, 0.2370, 0.6035, 0.2414, 0.5815,
		0.2443, 0.5728, 0.2487, 0.5625, 0.2545, 0.5509, 0.2589, 0.5435,
		0.2662, 0.5333, 0.2940, 0.5056, 0.3042, 0.4983, 0.3115, 0.4939,
		0.3247, 0.4866, 0.3466, 0.4793, 0.3612, 0.4763, 0.3714, 0.4749,
		0.6257, 0.4749, 0.6388, 0.4763, 0.6534, 0.4793, 0.6710, 0.4851,
		0.6812, 0.4895, 0.6870, 0.4924, 0.6943, 0.4968, 0.7046, 0.5041,
		0.7206, 0.5172, 0.7236, 0.5202, 0.7367, 0.5362, 0.7426, 0.5450,
		0.7469, 0.5523, 0.7499, 0.5582, 0.7542, 0.5684, 0.7601, 0.5830,
		0.7615, 0.5903, 0.7645, 0.6108, 0.7645, 0.6195, 0.7601, 0.6239,
		0.2399, 0.6239,
	},
	{
		0.3334, 0.0658, 0.3276, 0.0541, 0.3276, 0.0336, 0.3305, 0.0248,
		0.3378, 0.0146, 0.3407, 0.0117, 0.3510, 0.0044, 0.3597, 0.0015,
		0.3670, 0.0000, 0.6359, 0.0000, 0.6432, 0.0015, 0.6490, 0.0044,
		0.6563, 0.0088, 0.6666, 0.0190, 0.6724, 0.0307, 0.6739, 0.0380,
		0.6739, 0.0497, 0.6710, 0.0584, 0.6666, 0.0658, 0.6607, 0.0745,
		0.6593, 0.0760, 0.6447, 0.0847, 0.6373, 0.0862, 0.3627, 0.0862,
		0.3495, 0.0818,
	},
	{
		0.4675, 0.9667, 0.4539, 0.9467, 0.4505, 0.9401, 0.4489, 0.9351,
		0.4490, 0.6586, 0.4507, 0.6536, 0.4779, 0.6269, 0.5253, 0.6269,
		0.5541, 0.6552, 0.5539, 0.9383, 0.5505, 0.9450, 0.5455, 0.9533,
		0.5319, 0.9733, 0.5048, 1.0000, 0.5014, 1.0000,
	},
}
