package dev.mygo

import android.content.Context
import android.os.Handler
import android.os.Looper
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.PeriodicWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import org.json.JSONObject

/** WorkManager, each of the app's tasks a unique work of its name. */
class BackgroundPlugin : MyGoPlugin {
    init {
        instance = this
    }

    override fun call(method: String, args: JSONObject, reply: PluginReply) {
        val wm = WorkManager.getInstance(PluginHost.context)
        when (method) {
            "ready" -> {
                ready = true
                for (e in waiting) PluginHost.emit("background", "run", e)
                waiting.clear()
                reply.success(null)
            }
            "schedule" -> {
                val name = args.optString("name")
                val every = args.optDouble("every", 0.0)
                val delay = (args.optDouble("delay", 0.0) * 1000).toLong()
                val c = Constraints.Builder()
                    .setRequiredNetworkType(if (args.optBoolean("network")) NetworkType.CONNECTED else NetworkType.NOT_REQUIRED)
                    .setRequiresCharging(args.optBoolean("charging"))
                    .build()
                val data = Data.Builder().putString("name", name).build()
                if (every > 0) {
                    val r = PeriodicWorkRequest.Builder(MyGoWorker::class.java, (every * 1000).toLong(), TimeUnit.MILLISECONDS)
                        .setConstraints(c).setInputData(data).setInitialDelay(delay, TimeUnit.MILLISECONDS).build()
                    wm.enqueueUniquePeriodicWork(prefix + name, ExistingPeriodicWorkPolicy.CANCEL_AND_REENQUEUE, r)
                } else {
                    val r = OneTimeWorkRequest.Builder(MyGoWorker::class.java)
                        .setConstraints(c).setInputData(data).setInitialDelay(delay, TimeUnit.MILLISECONDS).build()
                    wm.enqueueUniqueWork(prefix + name, ExistingWorkPolicy.REPLACE, r)
                }
                reply.success(null)
            }
            "cancel" -> {
                wm.cancelUniqueWork(prefix + args.optString("name"))
                reply.success(null)
            }
            "done" -> {
                val id = args.optLong("run")
                val d = runs.remove(id)
                d?.invoke(args.optBoolean("ok"))
                reply.success(null)
            }
            else -> reply.failure(PluginFailure("UnknownMethod", method))
        }
    }

    companion object {
        private const val prefix = "mygo."
        internal var instance: BackgroundPlugin? = null
        private var ready = false
        private val waiting = ArrayList<Map<String, Any>>()
        private val runs = HashMap<Long, (Boolean) -> Unit>()
        private var nextRun = 1L

        /** Runs a task, on the UI thread: Go hears of it, once it listens. */
        internal fun run(name: String, done: (Boolean) -> Unit): Long {
            val id = nextRun++
            runs[id] = done
            val e = mapOf<String, Any>("run" to id, "name" to name)
            if (ready) PluginHost.emit("background", "run", e) else waiting.add(e)
            return id
        }

        internal fun stop(id: Long) {
            PluginHost.emit("background", "stop", mapOf("run" to id))
            runs.remove(id)
        }
    }
}

/** A task's turn, which may be what starts the process: Go starts with no Activity. */
class MyGoWorker(context: Context, params: WorkerParameters) : Worker(context, params) {
    @Volatile private var run = 0L
    private val main = Handler(Looper.getMainLooper())

    override fun doWork(): Result {
        val name = inputData.getString("name") ?: return Result.failure()
        val latch = CountDownLatch(1)
        var ok = false
        main.post {
            HostImpl.startHeadless(applicationContext)
            run = BackgroundPlugin.run(name) {
                ok = it
                latch.countDown()
            }
        }
        // WorkManager's limit is 10 minutes.
        if (!latch.await(9, TimeUnit.MINUTES)) {
            onStopped()
            return Result.retry()
        }
        return if (ok) Result.success() else Result.retry()
    }

    override fun onStopped() {
        val id = run
        if (id != 0L) main.post { BackgroundPlugin.stop(id) }
    }
}
