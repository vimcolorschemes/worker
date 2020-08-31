import printer

class Runner:
    def __init__(self, database, job):
        self.database = database
        self.last_job_at = self.database.get_last_job_at(job)

    def store_report(self, job, elapsed_time, result):
        self.database.create_report({"job": job, "elapsed_time": elapsed_time, **result})

        printer.success(f"{job} finished.")
        printer.info(f"elapsed_time: {elapsed_time}")
        for key in dict.keys(result):
            printer.info(f"{key}: {result[key]}")
