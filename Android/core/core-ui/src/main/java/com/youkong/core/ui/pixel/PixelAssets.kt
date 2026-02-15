package com.youkong.core.ui.pixel

import androidx.compose.ui.graphics.Color

// MARK: - Pixel Color Palette

object PixelPalette {
    val skin = Color(0xFFF5CC_A6u.toLong())
    val skinShadow = Color(0xFFD9AD85u.toLong())
    val hair = Color(0xFF332924u.toLong())
    val shirt = Color(0xFF4289D6u.toLong())
    val shirtShadow = Color(0xFF336BADu.toLong())
    val pants = Color(0xFF384260u.toLong())
    val pantsShadow = Color(0xFF283348u.toLong())
    val shoe = Color(0xFF473D38u.toLong())

    // Prop colors
    val food = Color(0xFFF28C40u.toLong())
    val laptop = Color(0xFF8F949Eu.toLong())
    val laptopScreen = Color(0xFF6BBFF2u.toLong())
    val book = Color(0xFFD94D40u.toLong())
    val phone = Color(0xFF59616Bu.toLong())
    val phoneScreen = Color(0xFF80D1F2u.toLong())
    val weights = Color(0xFF666B73u.toLong())
    val coffee = Color(0xFFB87A4Du.toLong())
    val coffeeBody = Color(0xFFEBE6D9u.toLong())
    val controller = Color(0xFF4D4D59u.toLong())
    val ball = Color(0xFFF28C26u.toLong())
    val bag = Color(0xFF8C5938u.toLong())
    val umbrella = Color(0xFFE6404Du.toLong())

    // Surface colors
    val wood = Color(0xFF9E7652u.toLong())
    val woodDark = Color(0xFF805C3Du.toLong())
    val fabric = Color(0xFF858C99u.toLong())
    val bedSheet = Color(0xFFD1D6E0u.toLong())
    val ground = Color(0xFF999E94u.toLong())

    // Expression colors
    val eyeWhite = Color.White
    val eyePupil = Color(0xFF261F1Au.toLong())
    val mouth = Color(0xFFD96B61u.toLong())
    val blush = Color(0xFFF2A699u.toLong())
}

// MARK: - Pixel Grid Type

typealias PixelGrid = List<List<Color?>>

// MARK: - Body Assets

object BodyAssets {
    private val P = PixelPalette

    val standing: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(null, P.pants, P.pants, null),
        listOf(null, P.pants, P.pants, null),
        listOf(P.shoe, null, null, P.shoe),
    )

    val standingFrame2: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(null, P.pants, P.pants, null),
        listOf(null, P.pantsShadow, P.pantsShadow, null),
        listOf(null, P.shoe, P.shoe, null),
    )

    val sitting: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(P.pants, P.pants, P.pants, P.pants),
        listOf(P.shoe, null, null, P.shoe),
    )

    val sittingFrame2: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.pants, P.pants, P.pants, P.pants),
        listOf(null, P.shoe, P.shoe, null),
    )

    val lying: PixelGrid = listOf(
        listOf(P.shirt, P.shirt, P.shirt, P.pants, P.pants, P.shoe),
        listOf(P.shirt, P.shirt, P.shirtShadow, P.pants, P.pantsShadow, P.shoe),
        listOf(null, null, null, null, null, null),
    )

    val lyingFrame2: PixelGrid = listOf(
        listOf(P.shirt, P.shirt, P.shirt, P.pants, P.pants, P.shoe),
        listOf(null, P.shirt, P.shirtShadow, P.pants, P.pantsShadow, P.shoe),
        listOf(null, null, null, null, null, null),
    )

    val running: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, null),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(P.pants, null, P.pants, null),
        listOf(null, null, null, P.pants),
        listOf(P.shoe, null, null, P.shoe),
    )

    val runningFrame2: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(null, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(null, P.pants, null, P.pants),
        listOf(P.pants, null, null, null),
        listOf(P.shoe, null, null, P.shoe),
    )

    val walking: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(null, P.pants, P.pants, null),
        listOf(P.pants, null, null, P.pants),
        listOf(P.shoe, null, null, P.shoe),
    )

    val walkingFrame2: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(null, P.pants, P.pants, null),
        listOf(null, P.pants, P.pants, null),
        listOf(null, P.shoe, P.shoe, null),
    )

    val crouching: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirtShadow, P.shirtShadow, null),
        listOf(P.pants, P.pants, P.pants, null),
        listOf(null, P.shoe, P.shoe, null),
    )

    val crouchingFrame2: PixelGrid = listOf(
        listOf(null, P.shirt, P.shirt, null),
        listOf(P.shirt, P.shirt, P.shirt, P.shirt),
        listOf(null, P.shirt, P.shirt, null),
        listOf(null, P.pants, P.pants, P.pants),
        listOf(null, P.shoe, P.shoe, null),
    )

    fun forPose(pose: String, frame: Int = 0): PixelGrid = when (pose) {
        "sitting" -> if (frame % 2 == 0) sitting else sittingFrame2
        "lying" -> if (frame % 2 == 0) lying else lyingFrame2
        "running" -> if (frame % 2 == 0) running else runningFrame2
        "walking" -> if (frame % 2 == 0) walking else walkingFrame2
        "crouching" -> if (frame % 2 == 0) crouching else crouchingFrame2
        else -> if (frame % 2 == 0) standing else standingFrame2
    }
}

// MARK: - Head Assets

object HeadAssets {
    private val P = PixelPalette

    val normal: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.skin, P.eyePupil, P.eyePupil, P.skin),
        listOf(null, P.skin, P.skin, null),
    )

    val happy: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.blush, P.eyePupil, P.eyePupil, P.blush),
        listOf(null, P.skin, P.mouth, null),
    )

    val sleepy: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.skin, P.skinShadow, P.skinShadow, P.skin),
        listOf(null, P.skin, P.skin, null),
    )

    val focused: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.skin, P.eyePupil, P.eyePupil, P.skin),
        listOf(null, P.skinShadow, P.skinShadow, null),
    )

    val excited: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.eyeWhite, P.eyePupil, P.eyePupil, P.eyeWhite),
        listOf(null, P.mouth, P.mouth, null),
    )

    val tired: PixelGrid = listOf(
        listOf(null, P.hair, P.hair, null),
        listOf(P.hair, P.hair, P.hair, P.hair),
        listOf(P.skin, P.skinShadow, P.skinShadow, P.skin),
        listOf(null, P.skinShadow, P.skin, null),
    )

    fun forExpression(expr: String): PixelGrid = when (expr) {
        "happy" -> happy
        "sleepy" -> sleepy
        "focused" -> focused
        "excited" -> excited
        "tired" -> tired
        else -> normal
    }
}

// MARK: - Arm Pixels

data class ArmPixel(val dx: Int, val dy: Int, val color: Color)

object ArmAssets {
    fun pixels(style: String, pose: String): List<ArmPixel> {
        if (pose == "lying") return emptyList()
        val P = PixelPalette
        return when (style) {
            "up" -> listOf(ArmPixel(-1, -1, P.skin), ArmPixel(4, -1, P.skin))
            "forward" -> listOf(ArmPixel(-1, 1, P.skin), ArmPixel(4, 1, P.skin))
            "holding" -> listOf(ArmPixel(-1, 1, P.skin), ArmPixel(4, 2, P.skin))
            "waving" -> listOf(ArmPixel(-1, 1, P.skin), ArmPixel(4, -1, P.skin), ArmPixel(5, -2, P.skin))
            "typing" -> listOf(ArmPixel(-1, 2, P.skin), ArmPixel(4, 2, P.skin))
            else -> listOf(ArmPixel(-1, 2, P.skin), ArmPixel(4, 2, P.skin)) // "down"
        }
    }
}

// MARK: - Prop Assets

data class PropData(val grid: PixelGrid, val offsetX: Int, val offsetY: Int)

object PropAssets {
    private val P = PixelPalette

    fun forProp(prop: String): PropData? = when (prop) {
        "food" -> PropData(listOf(listOf(P.food, P.food), listOf(P.coffeeBody, P.coffeeBody)), 5, 3)
        "laptop" -> PropData(listOf(listOf(P.laptopScreen, P.laptopScreen, P.laptopScreen), listOf(P.laptop, P.laptop, P.laptop)), 5, 2)
        "book" -> PropData(listOf(listOf(P.book, P.book), listOf(P.book, P.eyeWhite)), 5, 2)
        "phone" -> PropData(listOf(listOf(P.phoneScreen), listOf(P.phone)), 5, 1)
        "weights" -> PropData(listOf(listOf(P.weights, P.weights, P.weights)), -2, 4)
        "coffee" -> PropData(listOf(listOf(null, P.coffee), listOf(P.coffeeBody, P.coffeeBody)), 5, 3)
        "controller" -> PropData(listOf(listOf(P.controller, P.controller)), 5, 3)
        "ball" -> PropData(listOf(listOf(null, P.ball, null), listOf(P.ball, P.ball, P.ball), listOf(null, P.ball, null)), 5, 4)
        "bag" -> PropData(listOf(listOf(P.bag, P.bag), listOf(P.bag, P.bag), listOf(P.bag, P.bag)), -2, 2)
        "umbrella" -> PropData(listOf(listOf(P.umbrella, P.umbrella, P.umbrella), listOf(null, P.wood, null)), -1, -3)
        else -> null
    }
}

// MARK: - Surface Assets

object SurfaceAssets {
    private val P = PixelPalette

    fun forSurface(surface: String): PropData? = when (surface) {
        "chair" -> PropData(listOf(
            listOf(P.wood, null, null, null, null, P.wood),
            listOf(P.wood, P.wood, P.wood, P.wood, P.wood, P.wood),
            listOf(P.wood, null, null, null, null, P.wood),
            listOf(P.woodDark, null, null, null, null, P.woodDark),
        ), -1, 4)
        "desk" -> PropData(listOf(
            listOf(P.wood, P.wood, P.wood, P.wood, P.wood, P.wood, P.wood, P.wood),
            listOf(P.woodDark, null, null, null, null, null, null, P.woodDark),
        ), -2, 5)
        "bed" -> PropData(listOf(
            listOf(P.bedSheet, P.bedSheet, P.bedSheet, P.bedSheet, P.bedSheet, P.bedSheet, P.bedSheet, P.bedSheet),
            listOf(P.wood, P.wood, P.wood, P.wood, P.wood, P.wood, P.wood, P.wood),
        ), -2, 6)
        "couch" -> PropData(listOf(
            listOf(P.fabric, null, null, null, null, null, P.fabric),
            listOf(P.fabric, P.fabric, P.fabric, P.fabric, P.fabric, P.fabric, P.fabric),
            listOf(P.woodDark, null, null, null, null, null, P.woodDark),
        ), -1, 4)
        "ground" -> PropData(listOf(
            listOf(P.ground, P.ground, P.ground, P.ground, P.ground, P.ground),
        ), -1, 7)
        else -> null
    }
}
