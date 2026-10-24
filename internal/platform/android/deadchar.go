package android

import "unicode/utf8"

// accentToCombining maps the accents that Android's key character maps use for dead keys
// (the spacing accents and the combining marks themselves) to the combining mark that
// composes. A combining mark of U+0300 to U+036F maps to itself.
//
// Flutter: no counterpart (android.view.KeyCharacterMap.getDeadChar)
var accentToCombining = map[rune]rune{
	'`':    0x0300, // grave
	0x00b4: 0x0301, // acute
	'\'':   0x0301,
	'^':    0x0302, // circumflex
	'~':    0x0303, // tilde
	0x00af: 0x0304, // macron
	0x02d8: 0x0306, // breve
	0x02d9: 0x0307, // dot above
	0x00a8: 0x0308, // diaeresis
	'"':    0x0308,
	0x02da: 0x030a, // ring above
	0x02dd: 0x030b, // double acute
	0x02c7: 0x030c, // caron
	0x00b8: 0x0327, // cedilla
	0x02db: 0x0328, // ogonek
}

// deadChar returns the character that the accent and the base character c make, as
// android.view.KeyCharacterMap.getDeadChar does: the accent itself when c is the same
// accent or a space, the canonically composed character (Unicode NFC) when the pair
// has one, and 0 otherwise. The accent is a spacing accent or a combining mark.
//
// Deviation: Android composes with java.text.Normalizer, which this module does not
// have (it has no golang.org/x/text); composedCharacters holds the compositions of the
// Latin, Greek and Cyrillic letters and of the additional Latin letters of Unicode 16
// (SPEC §12.10).
//
// Flutter: no counterpart (android.view.KeyCharacterMap.getDeadChar)
func deadChar(accent, c rune) rune {
	if c == accent || c == ' ' {
		return accent
	}
	mark := accent
	if m, ok := accentToCombining[accent]; ok {
		mark = m
	}
	pairs, ok := composedCharacters[mark]
	if !ok {
		return 0
	}
	// pairs holds base and composed character, one after the other.
	var prev rune
	isBase := true
	for _, r := range pairs {
		if isBase {
			prev = r
		} else if prev == c {
			return r
		}
		isBase = !isBase
	}
	return 0
}

// validRune reports whether r can be put in a string.
func validRune(r rune) bool { return r > 0 && utf8.ValidRune(r) }

// composedCharacters maps a combining mark to the pairs "base, composed character" that
// Unicode NFC composes with it.
//
// Flutter: no counterpart (java.text.Normalizer)
var composedCharacters = map[rune]string{
	0x0300: "AÀEÈIÌNǸOÒUÙWẀYỲaàeèiìnǹoòuùwẁyỳ¨῭ÂẦÊỀÔỒÜǛâầêềôồüǜĂẰăằĒḔēḕŌṐōṑƠỜơờƯỪưừΑᾺΕῈΗῊΙῚΟῸΥῪΩῺαὰεὲηὴιὶοὸυὺωὼϊῒϋῢЕЀИЍеѐиѝἀἂἁἃἈἊἉἋἐἒἑἓἘἚἙἛἠἢἡἣἨἪἩἫἰἲἱἳἸἺἹἻὀὂὁὃὈὊὉὋὐὒὑὓὙὛὠὢὡὣὨὪὩὫᾀᾂᾁᾃᾈᾊᾉᾋᾐᾒᾑᾓᾘᾚᾙᾛᾠᾢᾡᾣᾨᾪᾩᾫᾳᾲιὶ᾿῍ῃῂῳῲ῾῝",
	0x0301: "AÁCĆEÉGǴIÍKḰLĹMḾNŃOÓPṔRŔSŚUÚWẂYÝZŹaácćeégǵiíkḱlĺmḿnńoópṕrŕsśuúwẃyýzź¨΅ÂẤÅǺÆǼÇḈÊẾÏḮÔỐÕṌØǾÜǗâấåǻæǽçḉêếïḯôốõṍøǿüǘĂẮăắĒḖēḗŌṒōṓŨṸũṹƠỚơớƯỨưứΑΆΕΈΗΉΙΊΟΌΥΎΩΏαάεέηήιίοόυύωώϊΐϋΰϒϓГЃКЌгѓкќἀἄἁἅἈἌἉἍἐἔἑἕἘἜἙἝἠἤἡἥἨἬἩἭἰἴἱἵἸἼἹἽὀὄὁὅὈὌὉὍὐὔὑὕὙὝὠὤὡὥὨὬὩὭᾀᾄᾁᾅᾈᾌᾉᾍᾐᾔᾑᾕᾘᾜᾙᾝᾠᾤᾡᾥᾨᾬᾩᾭᾳᾴιί᾿῎ῃῄῳῴ῾῞",
	0x0302: "AÂCĈEÊGĜHĤIÎJĴOÔSŜUÛWŴYŶZẐaâcĉeêgĝhĥiîjĵoôsŝuûwŵyŷzẑẠẬạậẸỆẹệỌỘọộ",
	0x0303: "AÃEẼIĨNÑOÕUŨVṼYỸaãeẽiĩnñoõuũvṽyỹÂẪÊỄÔỖâẫêễôỗĂẴăẵƠỠơỡƯỮưữ",
	0x0304: "AĀEĒGḠIĪOŌUŪYȲaāeēgḡiīoōuūyȳÄǞÆǢÕȬÖȪÜǕäǟæǣõȭöȫüǖǪǬǫǭȦǠȧǡȮȰȯȱΑᾹΙῙΥῩαᾱιῑυῡИӢУӮиӣуӯḶḸḷḹṚṜṛṝιῑ",
	0x0306: "AĂEĔGĞIĬOŎUŬaăeĕgğiĭoŏuŭȨḜȩḝΑᾸΙῘΥῨαᾰιῐυῠАӐЕӖЖӁИЙУЎаӑеӗжӂийуўẠẶạặιῐ",
	0x0307: "AȦBḂCĊDḊEĖFḞGĠHḢIİMṀNṄOȮPṖRṘSṠTṪWẆXẊYẎZŻaȧbḃcċdḋeėfḟgġhḣmṁnṅoȯpṗrṙsṡtṫwẇxẋyẏzżŚṤśṥŠṦšṧſẛṢṨṣṩ",
	0x0308: "AÄEËHḦIÏOÖUÜWẄXẌYŸaäeëhḧiïoötẗuüwẅxẍyÿÕṎõṏŪṺūṻΙΪΥΫιϊυϋϒϔІЇАӒЕЁЖӜЗӞИӤОӦУӰЧӴЫӸЭӬаӓеёжӝзӟиӥоӧуӱчӵыӹэӭіїӘӚәӛӨӪөӫιϊ",
	0x0309: "AẢEẺIỈOỎUỦYỶaảeẻiỉoỏuủyỷÂẨÊỂÔỔâẩêểôổĂẲăẳƠỞơởƯỬưử",
	0x030a: "AÅUŮaåuůwẘyẙ",
	0x030b: "OŐUŰoőuűУӲуӳ",
	0x030c: "AǍCČDĎEĚGǦHȞIǏKǨLĽNŇOǑRŘSŠTŤUǓZŽaǎcčdďeěgǧhȟiǐjǰkǩlľnňoǒrřsštťuǔzžÜǙüǚƷǮ",
	0x030f: "AȀEȄIȈOȌRȐUȔaȁeȅiȉoȍrȑuȕѴѶѵѷ",
	0x0311: "AȂEȆIȊOȎRȒUȖaȃeȇiȋoȏrȓuȗ",
	0x0313: "ΑἈΕἘΗἨΙἸΟὈΩὨαἀεἐηἠιἰοὀρῤυὐωὠᾳᾀᾼᾈιἰῃᾐῌᾘῳᾠῼᾨ",
	0x0314: "ΑἉΕἙΗἩΙἹΟὉΡῬΥὙΩὩαἁεἑηἡιἱοὁρῥυὑωὡᾳᾁᾼᾉιἱῃᾑῌᾙῳᾡῼᾩ",
	0x031b: "OƠUƯoơuưÒỜÓỚÕỠÙỪÚỨòờóớõỡùừúứŨỮũữỌỢọợỎỞỏởỤỰụựỦỬủử",
	0x0323: "AẠBḄDḌEẸHḤIỊKḲLḶMṂNṆOỌRṚSṢTṬUỤVṾWẈYỴZẒaạbḅdḍeẹhḥiịkḳlḷmṃnṇoọrṛsṣtṭuụvṿwẉyỵzẓÂẬÊỆÔỘâậêệôộĂẶăặƠỢơợƯỰưựṠṨṡṩ",
	0x0324: "UṲuṳ",
	0x0325: "AḀaḁ",
	0x0326: "SȘTȚsștț",
	0x0327: "CÇDḐEȨGĢHḨKĶLĻNŅRŖSŞTŢcçdḑeȩgģhḩkķlļnņrŗsştţĆḈćḉĔḜĕḝ",
	0x0328: "AĄEĘIĮOǪUŲaąeęiįoǫuųŌǬōǭ",
	0x032d: "DḒEḘLḼNṊTṰUṶdḓeḙlḽnṋtṱuṷ",
	0x032e: "HḪhḫ",
	0x0330: "EḚIḬUṴeḛiḭuṵ",
	0x0331: "BḆDḎKḴLḺNṈRṞTṮZẔbḇdḏhẖkḵlḻnṉrṟtṯzẕ",
	0x0338: "<≮=≠>≯",
	0x0340: "AÀEÈIÌNǸOÒUÙWẀYỲaàeèiìnǹoòuùwẁyỳ¨῭ÂẦÊỀÔỒÜǛâầêềôồüǜĂẰăằĒḔēḕŌṐōṑƠỜơờƯỪưừΑᾺΕῈΗῊΙῚΟῸΥῪΩῺαὰεὲηὴιὶοὸυὺωὼϊῒϋῢЕЀИЍеѐиѝἀἂἁἃἈἊἉἋἐἒἑἓἘἚἙἛἠἢἡἣἨἪἩἫἰἲἱἳἸἺἹἻὀὂὁὃὈὊὉὋὐὒὑὓὙὛὠὢὡὣὨὪὩὫᾀᾂᾁᾃᾈᾊᾉᾋᾐᾒᾑᾓᾘᾚᾙᾛᾠᾢᾡᾣᾨᾪᾩᾫᾳᾲιὶ᾿῍ῃῂῳῲ῾῝",
	0x0341: "AÁCĆEÉGǴIÍKḰLĹMḾNŃOÓPṔRŔSŚUÚWẂYÝZŹaácćeégǵiíkḱlĺmḿnńoópṕrŕsśuúwẃyýzź¨΅ÂẤÅǺÆǼÇḈÊẾÏḮÔỐÕṌØǾÜǗâấåǻæǽçḉêếïḯôốõṍøǿüǘĂẮăắĒḖēḗŌṒōṓŨṸũṹƠỚơớƯỨưứΑΆΕΈΗΉΙΊΟΌΥΎΩΏαάεέηήιίοόυύωώϊΐϋΰϒϓГЃКЌгѓкќἀἄἁἅἈἌἉἍἐἔἑἕἘἜἙἝἠἤἡἥἨἬἩἭἰἴἱἵἸἼἹἽὀὄὁὅὈὌὉὍὐὔὑὕὙὝὠὤὡὥὨὬὩὭᾀᾄᾁᾅᾈᾌᾉᾍᾐᾔᾑᾕᾘᾜᾙᾝᾠᾤᾡᾥᾨᾬᾩᾭᾳᾴιί᾿῎ῃῄῳῴ῾῞",
	0x0342: "¨῁αᾶηῆιῖυῦωῶϊῗϋῧἀἆἁἇἈἎἉἏἠἦἡἧἨἮἩἯἰἶἱἷἸἾἹἿὐὖὑὗὙὟὠὦὡὧὨὮὩὯᾀᾆᾁᾇᾈᾎᾉᾏᾐᾖᾑᾗᾘᾞᾙᾟᾠᾦᾡᾧᾨᾮᾩᾯᾳᾷιῖ᾿῏ῃῇῳῷ῾῟",
	0x0343: "ΑἈΕἘΗἨΙἸΟὈΩὨαἀεἐηἠιἰοὀρῤυὐωὠᾳᾀᾼᾈιἰῃᾐῌᾘῳᾠῼᾨ",
	0x0344: "IḮUǗiḯuǘιΐυΰιΐ",
	0x0345: "ΑᾼΗῌΩῼάᾴήῄαᾳηῃωῳώῴἀᾀἁᾁἂᾂἃᾃἄᾄἅᾅἆᾆἇᾇἈᾈἉᾉἊᾊἋᾋἌᾌἍᾍἎᾎἏᾏἠᾐἡᾑἢᾒἣᾓἤᾔἥᾕἦᾖἧᾗἨᾘἩᾙἪᾚἫᾛἬᾜἭᾝἮᾞἯᾟὠᾠὡᾡὢᾢὣᾣὤᾤὥᾥὦᾦὧᾧὨᾨὩᾩὪᾪὫᾫὬᾬὭᾭὮᾮὯᾯὰᾲάᾴὴῂήῄὼῲώῴᾶᾷῆῇῶῷ",
}
